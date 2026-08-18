package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"telegramBittorrentDownloader/service"
	"telegramBittorrentDownloader/types"
	"time"

	tele "gopkg.in/telebot.v4"
)

const (
	itemsPerPage   = 5  // 每页显示的条目数，减少以适应 400 字符限制
	titleMaxLength = 50 // 标题最大长度，超过部分截断
)

func InitBot(ctx context.Context, config *types.Config, service *service.Service) {
	pref := tele.Settings{
		Token:  config.Bot.Token,
		Client: config.Proxy.Client,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create bot", "error", err)
		return
	}
	slog.InfoContext(ctx, "Bot started", "bot_name", config.Bot.BotName)

	whiteListOnly := b.Group()
	if config.SenderID != nil && len(config.SenderID) > 0 {
		whiteListOnly.Use(func(next tele.HandlerFunc) tele.HandlerFunc {
			return func(c tele.Context) error {
				if c.Sender() == nil {
					return next(c)
				}
				if err := checkUser(config, c.Sender().ID); err != nil {
					if c.Callback() != nil {
						_ = c.Respond(&tele.CallbackResponse{Text: fmt.Sprintf("%v 你好，你不在白名单中，无法使用该 bot，你的 ID 为 %d", c.Sender().Username, c.Sender().ID)})
						return nil
					}
					return c.Send(fmt.Sprintf("%v 你好，你不在白名单中，无法使用该 bot，你的 ID 为 %d", c.Sender().Username, c.Sender().ID))
				}
				return next(c)
			}
		})
	}

	// 处理搜索命令
	whiteListOnly.Handle("/nyaa", func(c tele.Context) error {
		query := c.Message().Payload
		query = strings.TrimSpace(query)
		if query == "" {
			return c.Send("请输入搜索关键词")
		}
		return handleSearch(ctx, c, service, query, 0, config.Bot.BotName)
	})

	whiteListOnly.Handle("/qb", func(c tele.Context) error {
		magnet := c.Message().Payload
		err := addMagnet(ctx, magnet, service)
		if err != nil {
			return c.Send(fmt.Sprintf("添加下载失败: %s", err.Error()))
		}
		return c.Send("✅ 已成功添加到 qBittorrent")
	})

	whiteListOnly.Handle("/start", func(c tele.Context) error {
		magnet := c.Message().Payload
		magnet = strings.TrimSpace(magnet)
		if magnet != "" {
			if isStartDownloadPayload(magnet) {
				if len(magnet) == 40 {
					magnetLink, _ := service.Cache.Get(magnet)
					if magnetLink != "" {
						slog.InfoContext(ctx, "命中缓存", "magnet", magnet, "magnetLink", magnetLink)
						magnet = magnetLink
					}
				}
				err := addMagnet(ctx, magnet, service)
				if err != nil {
					return c.Send(fmt.Sprintf("添加下载失败: %s", err.Error()))
				}
				return c.Send("✅ 已成功添加到 qBittorrent")
			}
			return handleSearch(ctx, c, service, magnet, 0, config.Bot.BotName)
		}
		return c.Send("欢迎使用 Bittorrent 下载器！\n使用 /nyaa <关键词> 搜索 torrents。\n使用 /qb <磁力链接> 添加下载到 qBittorrent。")
	})

	// 处理下载回调
	whiteListOnly.Handle(&tele.InlineButton{Unique: "dl_qb"}, func(c tele.Context) error {
		defer c.Respond(&tele.CallbackResponse{Text: "正在添加到 qBittorrent..."})
		magnetHash := c.Callback().Data
		magnet := "magnet:?xt=urn:btih:" + magnetHash

		dl, ok := service.Downloader["qbittorrent"]
		if !ok || dl == nil {
			return c.Send("❌ 错误：qBittorrent 下载器未配置或初始化失败")
		}

		err := dl.AddMagnet(ctx, magnet)
		if err != nil {
			return c.Send(fmt.Sprintf("❌ 添加下载失败: %s", err.Error()))
		}
		return c.Send(fmt.Sprintf("✅ 任务已添加: <code>%s</code>", magnetHash), tele.ModeHTML)
	})

	// 处理翻页回调
	whiteListOnly.Handle(&tele.InlineButton{Unique: "prev_page"}, func(c tele.Context) error {
		defer c.Respond()
		data := c.Callback().Data
		parts := strings.Split(data, "|")
		if len(parts) != 2 {
			return nil
		}
		query := parts[0]
		page, _ := strconv.Atoi(parts[1])
		if page > 0 {
			page--
		}
		return handleSearch(ctx, c, service, query, page, config.Bot.BotName)
	})

	whiteListOnly.Handle(&tele.InlineButton{Unique: "next_page"}, func(c tele.Context) error {
		defer c.Respond()
		data := c.Callback().Data
		parts := strings.Split(data, "|")
		if len(parts) != 2 {
			return nil
		}
		query := parts[0]
		page, _ := strconv.Atoi(parts[1])
		page++
		return handleSearch(ctx, c, service, query, page, config.Bot.BotName)
	})

	b.Start()
}

func isStartDownloadPayload(payload string) bool {
	return len(payload) == 40 || strings.HasPrefix(payload, "magnet:?")
}

func addMagnet(ctx context.Context, magnet string, service *service.Service) error {
	magnet = strings.TrimSpace(magnet)
	if magnet == "" {
		return fmt.Errorf("磁力链接不能为空")
	}

	if !strings.HasPrefix(magnet, "magnet:?") {
		// 如果只是 hash，尝试补全
		if len(magnet) == 40 {
			magnet = "magnet:?xt=urn:btih:" + magnet
		} else {
			return fmt.Errorf("无效的磁力链接或 Hash")
		}
	}

	dl, ok := service.Downloader["qbittorrent"]
	if !ok || dl == nil {
		return fmt.Errorf("qBittorrent 下载器未配置或初始化失败")
	}

	err := dl.AddMagnet(ctx, magnet)
	return err
}

// 处理搜索和翻页逻辑
func handleSearch(ctx context.Context, c tele.Context, service *service.Service, query string, page int, botName string) error {
	query = strings.ReplaceAll(query, " ", "+")
	slog.InfoContext(ctx, "Searching for torrents", "query", query, "page", page)

	s, ok := service.Searcher["nyaa"]
	if !ok || s == nil {
		return c.Send("❌ 错误：Nyaa 搜索器未配置或初始化失败")
	}

	result, err := s.Search(ctx, query)
	if err != nil {
		return c.Send(fmt.Sprintf("搜索失败: %s", err.Error()))
	}

	if len(result.Data) == 0 {
		return c.Send("没有找到相关结果")
	}

	// 计算分页
	totalPages := (len(result.Data) + itemsPerPage - 1) / itemsPerPage
	start := page * itemsPerPage
	end := start + itemsPerPage
	if end > len(result.Data) {
		end = len(result.Data)
	}

	// 构建消息内容与下载按钮
	var msg strings.Builder
	var keyboard [][]tele.InlineButton
	var dlRow []tele.InlineButton

	// Escape query for HTML
	safeQuery := strings.ReplaceAll(query, "&", "&amp;")
	safeQuery = strings.ReplaceAll(safeQuery, "<", "&lt;")
	safeQuery = strings.ReplaceAll(safeQuery, ">", "&gt;")

	msg.WriteString(fmt.Sprintf("搜索内容: %s\n", safeQuery))
	msg.WriteString(fmt.Sprintf("第 %d/%d 页 (共 %d 个结果)\n\n", page+1, totalPages, len(result.Data)))

	for i := start; i < end; i++ {
		torrent := result.Data[i]

		title := torrent.Title
		runes := []rune(title)
		if len(runes) > titleMaxLength {
			title = string(runes[:titleMaxLength]) + "..."
		}

		// Escape title
		title = strings.ReplaceAll(title, "&", "&amp;")
		title = strings.ReplaceAll(title, "<", "&lt;")
		title = strings.ReplaceAll(title, ">", "&gt;")

		// Extract hash for callback and display
		magnet := torrent.Magnet
		if magnet == "" {
			continue
		}
		hash := ""
		if startIdx := strings.Index(magnet, "btih:"); startIdx != -1 {
			hash = magnet[startIdx+5:]
			if endIdx := strings.Index(hash, "&"); endIdx != -1 {
				hash = hash[:endIdx]
			}
		}
		if hash == "" {
			continue
		}
		err = service.Cache.Set(hash, magnet)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to set cache", "hash", hash, "magnet", magnet, "error", err)
		}

		msg.WriteString(fmt.Sprintf("📌 %s\n", title))
		msg.WriteString(fmt.Sprintf("📦 %s | 👤 %d | ⏬ %d\n", torrent.Size, torrent.Seeders, torrent.Downloads))
		msg.WriteString(fmt.Sprintf("<a href=\"https://t.me/%s?&start=%s\">点击添加下载</a>\n\n", botName, hash))

		// 为每个结果添加下载按钮
		dlRow = append(dlRow, tele.InlineButton{
			Unique: "dl_qb",
			Text:   fmt.Sprintf("📥 下载 %d", i-start+1),
			Data:   hash,
		})
	}

	if len(dlRow) > 0 {
		keyboard = append(keyboard, dlRow)
	}

	// 构建翻页键盘
	if totalPages > 1 {
		var row []tele.InlineButton

		if page > 0 {
			prevData := fmt.Sprintf("%s|%d", query, page)
			row = append(row, tele.InlineButton{
				Unique: "prev_page",
				Text:   "⬅️ 上一页",
				Data:   prevData,
			})
		}

		if page < totalPages-1 {
			nextData := fmt.Sprintf("%s|%d", query, page)
			row = append(row, tele.InlineButton{
				Unique: "next_page",
				Text:   "下一页 ➡️",
				Data:   nextData,
			})
		}

		if len(row) > 0 {
			keyboard = append(keyboard, row)
		}
	}

	options := &tele.SendOptions{
		ReplyMarkup: &tele.ReplyMarkup{
			InlineKeyboard: keyboard,
		},
		DisableWebPagePreview: true,
		ParseMode:             tele.ModeHTML,
	}

	if c.Callback() != nil {
		return c.Edit(msg.String(), options)
	}
	return c.Send(msg.String(), options)
}

func checkUser(config *types.Config, senderId int64) error {
	if len(config.SenderID) == 0 {
		return nil
	}
	for _, id := range config.SenderID {
		if id == senderId {
			return nil
		}
	}
	return fmt.Errorf("用户 %d 未被授权使用该 bot", senderId)
}
