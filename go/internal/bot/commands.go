package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"

	"github.com/varubogu/discord_message_cleaner_go/internal/discordapi"
	"github.com/varubogu/discord_message_cleaner_go/internal/discordurl"
	"github.com/varubogu/discord_message_cleaner_go/internal/i18n"
	"github.com/varubogu/discord_message_cleaner_go/internal/lifetime"
	"github.com/varubogu/discord_message_cleaner_go/internal/registry"
	"github.com/varubogu/discord_message_cleaner_go/internal/store"
)

// コマンドで指定できるチャンネルの種類（Python 版の TextChannel | VoiceChannel | Thread 相当）
var targetChannelTypes = []discord.ChannelType{
	discord.ChannelTypeGuildText,
	discord.ChannelTypeGuildNews,
	discord.ChannelTypeGuildVoice,
	discord.ChannelTypeGuildStageVoice,
	discord.ChannelTypeGuildPublicThread,
	discord.ChannelTypeGuildPrivateThread,
	discord.ChannelTypeGuildNewsThread,
}

func commandDefinitions() []discord.ApplicationCommandCreate {
	guildOnly := []discord.InteractionContextType{discord.InteractionContextTypeGuild}
	channelOpt := func(desc string, required bool) discord.ApplicationCommandOption {
		return discord.ApplicationCommandOptionChannel{
			Name:         "channel",
			Description:  desc,
			Required:     required,
			ChannelTypes: targetChannelTypes,
		}
	}
	messageURLOpt := discord.ApplicationCommandOptionString{
		Name:        "message_url",
		Description: "メッセージURLを指定します。",
		Required:    true,
	}
	return []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{
			Name:        "enable",
			Description: "監視対象のチャンネルを設定します。必須ロール:message_cleaner_admin",
			Contexts:    guildOnly,
			Options: []discord.ApplicationCommandOption{
				channelOpt("削除対象チャンネル", true),
				discord.ApplicationCommandOptionString{
					Name:        "lifetime",
					Description: "どれだけの期間メッセージを残すか（省略時: 1day）",
				},
			},
		},
		discord.SlashCommandCreate{
			Name:        "disable",
			Description: "チャンネルを監視対象から解除します。必須ロール:message_cleaner_admin",
			Contexts:    guildOnly,
			Options:     []discord.ApplicationCommandOption{channelOpt("監視解除対象チャンネル", true)},
		},
		discord.SlashCommandCreate{
			Name:        "exclusion_add",
			Description: "メッセージURLを指定することで削除設定から除外することができます。必須ロール:message_cleaner_admin",
			Contexts:    guildOnly,
			Options:     []discord.ApplicationCommandOption{messageURLOpt},
		},
		discord.SlashCommandCreate{
			Name:        "exclusion_remove",
			Description: "設定済みの除外設定を解除します。必須ロール:message_cleaner_admin",
			Contexts:    guildOnly,
			Options:     []discord.ApplicationCommandOption{messageURLOpt},
		},
		discord.SlashCommandCreate{
			Name:        "settings",
			Description: "現在のサーバーで設定中の設定内容を表示します。必須ロール:message_cleaner_admin",
			Contexts:    guildOnly,
			Options:     []discord.ApplicationCommandOption{channelOpt("省略した場合、サーバー内の全ての設定が表示されます。", false)},
		},
		discord.SlashCommandCreate{
			Name:        "clear",
			Description: "対象したチャンネルのメッセージをすべて削除します。必須ロール:message_cleaner_admin",
			Contexts:    guildOnly,
			Options:     []discord.ApplicationCommandOption{channelOpt("削除対象チャンネル", true)},
		},
		discord.SlashCommandCreate{
			Name:        "help",
			Description: "Display bot usage and command list.",
		},
	}
}

func (b *Bot) onCommand(e *events.ApplicationCommandInteractionCreate) {
	data := e.SlashCommandInteractionData()
	defer func() {
		if r := recover(); r != nil {
			b.log.Error("panic in command handler", slog.String("command", data.CommandName()), slog.Any("panic", r))
			b.reply(e, b.msgs.Display(i18n.Unknown))
		}
	}()

	if data.CommandName() == "help" {
		b.cmdHelp(e)
		return
	}
	if !b.checkAdmin(e) {
		return
	}
	switch data.CommandName() {
	case "enable":
		b.cmdEnable(e, data)
	case "disable":
		b.cmdDisable(e, data)
	case "exclusion_add":
		b.cmdExclusionAdd(e, data)
	case "exclusion_remove":
		b.cmdExclusionRemove(e, data)
	case "settings":
		b.cmdSettings(e, data)
	case "clear":
		b.cmdClear(e, data)
	}
}

func (b *Bot) onComponent(e *events.ComponentInteractionCreate) {
	b.confirms.handle(b.ctx, e)
}

// ---------------------------------------------------------------------------
// 共通処理
// ---------------------------------------------------------------------------

func (b *Bot) reply(e *events.ApplicationCommandInteractionCreate, content string) {
	err := e.CreateMessage(discord.NewMessageCreate().
		WithContent(content).
		WithEphemeral(true).
		WithAllowedMentions(&discord.AllowedMentions{}))
	if err != nil {
		b.log.Error("failed to reply", slog.Any("err", err))
	}
}

// followup は DeferCreateMessage 後の応答を送る。
func (b *Bot) followup(e *events.ApplicationCommandInteractionCreate, update discord.MessageUpdate) {
	if _, err := b.client.Rest.UpdateInteractionResponse(e.ApplicationID(), e.Token(),
		update.WithAllowedMentions(&discord.AllowedMentions{})); err != nil {
		b.log.Error("failed to send followup", slog.Any("err", err))
	}
}

// checkAdmin はコマンド実行者が管理ロール（message_cleaner_admin）を持つか確認する。
func (b *Bot) checkAdmin(e *events.ApplicationCommandInteractionCreate) bool {
	member := e.Member()
	guildID := e.GuildID()
	if member == nil || guildID == nil {
		b.reply(e, "サーバー内で実行してください。")
		return false
	}
	for _, roleID := range member.RoleIDs {
		if role, ok := b.client.Caches.Role(*guildID, roleID); ok && role.Name == b.cfg.AdminRoleName {
			return true
		}
	}
	b.reply(e, b.msgs.Display(i18n.NoBotUsagePermission))
	return false
}

// readPermCodes はメッセージを読む権限の不足を返す。
func readPermCodes(p discord.Permissions) []i18n.Code {
	if p.Has(discord.PermissionAdministrator) {
		return nil
	}
	if p.Missing(discord.PermissionViewChannel) {
		return []i18n.Code{i18n.MessageReadPermDenied}
	}
	if p.Missing(discord.PermissionReadMessageHistory) {
		return []i18n.Code{i18n.MessageReadHistoryPermDenied}
	}
	return nil
}

// deletePermCodes はメッセージを削除する権限の不足を返す。
func deletePermCodes(p discord.Permissions) []i18n.Code {
	if p.Has(discord.PermissionAdministrator) {
		return nil
	}
	codes := readPermCodes(p)
	if p.Missing(discord.PermissionManageMessages) {
		codes = append(codes, i18n.MessageDeletePermDenied)
	}
	return codes
}

func (b *Bot) codesMessage(codes []i18n.Code) string {
	lines := make([]string, len(codes))
	for i, c := range codes {
		lines[i] = b.msgs.Display(c)
	}
	return strings.Join(lines, "\n")
}

func (b *Bot) logCodes(codes []i18n.Code) string {
	lines := make([]string, len(codes))
	for i, c := range codes {
		lines[i] = b.msgs.Log(c)
	}
	return strings.Join(lines, " / ")
}

// checkChannelPerms はコマンド実行者と Bot の両方がチャンネルで必要な権限を持つか確認する。
func (b *Bot) checkChannelPerms(e *events.ApplicationCommandInteractionCreate, ch discord.ResolvedChannel, needDelete bool) bool {
	check := readPermCodes
	if needDelete {
		check = deletePermCodes
	}
	if codes := check(ch.Permissions); len(codes) > 0 {
		b.log.Info("user lacks permission", slog.Uint64("channel_id", uint64(ch.ID)), slog.String("user", e.User().Username), slog.String("reason", b.logCodes(codes)))
		b.reply(e, b.codesMessage(codes))
		return false
	}
	// Bot 自身の権限。権限不足のまま監視すると 403 を繰り返すことになるため事前に弾く。
	if needDelete && ch.AppPermissions != 0 {
		if codes := deletePermCodes(ch.AppPermissions); len(codes) > 0 {
			b.log.Info("bot lacks permission", slog.Uint64("channel_id", uint64(ch.ID)), slog.String("reason", b.logCodes(codes)))
			b.reply(e, fmt.Sprintf("Botに%sの権限が不足しています。\n%s", mention(ch.ID), b.codesMessage(codes)))
			return false
		}
	}
	return true
}

func mention(channelID snowflake.ID) string {
	return discord.ChannelMention(channelID)
}

func (b *Bot) lang() string { return b.msgs.Lang() }

// ---------------------------------------------------------------------------
// /enable
// ---------------------------------------------------------------------------

func (b *Bot) cmdEnable(e *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	ch := data.Channel("channel")
	lifetimeStr, ok := data.OptString("lifetime")
	if !ok {
		lifetimeStr = "1day"
	}
	lt, err := lifetime.Parse(lifetimeStr)
	if err != nil {
		b.reply(e, err.Error())
		return
	}
	if !b.checkChannelPerms(e, ch, true) {
		return
	}

	ltText := lifetime.Format(lt, b.lang())
	var msg string
	if cur, ok := b.reg.Get(ch.ID); ok {
		msg = fmt.Sprintf("%sのライフタイムを[%s]から[%s]に変更しますか？\n記録済みのメッセージも新しいライフタイムで削除予定日時を再計算します。",
			mention(ch.ID), lifetime.Format(cur.Lifetime, b.lang()), ltText)
	} else {
		msg = fmt.Sprintf("%sを監視対象にしますか？\n監視対象のチャンネルのメッセージは[%s]が経過すると削除されます。", mention(ch.ID), ltText)
	}
	msg += fmt.Sprintf("\n(続行する場合、%d秒以内に押してください)", int(b.cfg.ConfirmTimeout.Seconds()))

	guildID := *e.GuildID()
	b.confirms.ask(e, msg, func(ctx context.Context) string {
		// 既存メッセージを取り込まない場合は、現時点の最新メッセージ以降のみを対象にする
		var syncFrom snowflake.ID
		if !b.cfg.BackfillOnEnable && ch.LastMessageID != nil {
			syncFrom = *ch.LastMessageID
		}
		res, err := b.store.UpsertChannel(ctx, guildID, ch.ID, lt, syncFrom)
		if err != nil {
			b.log.Error("failed to enable channel", slog.Any("err", err))
			return b.msgs.Display(i18n.Unknown)
		}
		b.reg.Set(registry.Entry{GuildID: guildID, ChannelID: ch.ID, Lifetime: lt})
		b.log.Info("channel enabled", slog.Uint64("guild_id", uint64(guildID)), slog.Uint64("channel_id", uint64(ch.ID)),
			slog.Duration("lifetime", lt), slog.Bool("created", res.Created), slog.Int64("rescheduled", res.Rescheduled))

		out := fmt.Sprintf("[%s]に定期削除を設定しました。\nライフタイム[%s]", mention(ch.ID), ltText)
		if res.Created {
			if b.cfg.BackfillOnEnable {
				out += "\n既存のメッセージを取り込み中です。取り込み完了後、期限切れのものから順次削除されます。"
			}
			// 新規時は既存メッセージを取り込む（無効時でも syncFrom 以降の取りこぼし回収になる）
			b.syncer.Start(b.ctx, guildID, ch.ID)
		} else {
			out += fmt.Sprintf("\n記録済みメッセージ%d件の削除予定日時を更新しました。", res.Rescheduled)
			b.deleter.Wake()
		}
		return out
	})
}

// ---------------------------------------------------------------------------
// /disable
// ---------------------------------------------------------------------------

func (b *Bot) cmdDisable(e *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	ch := data.Channel("channel")
	// 解除は Bot の権限が不足していても行えるよう、実行者の権限のみ確認する
	if codes := deletePermCodes(ch.Permissions); len(codes) > 0 {
		b.reply(e, b.codesMessage(codes))
		return
	}
	guildID := *e.GuildID()
	existed, err := b.store.DeleteChannel(b.ctx, guildID, ch.ID)
	if err != nil {
		b.log.Error("failed to disable channel", slog.Any("err", err))
		b.reply(e, b.msgs.Display(i18n.Unknown))
		return
	}
	b.reg.Delete(ch.ID)
	b.deleter.DropChannel(ch.ID)
	if !existed {
		b.reply(e, fmt.Sprintf("%sは削除監視対象ではありません。", mention(ch.ID)))
		return
	}
	b.log.Info("channel disabled", slog.Uint64("guild_id", uint64(guildID)), slog.Uint64("channel_id", uint64(ch.ID)))
	b.reply(e, fmt.Sprintf("%sの設定を解除しました", mention(ch.ID)))
}

// ---------------------------------------------------------------------------
// /exclusion_add, /exclusion_remove
// ---------------------------------------------------------------------------

// resolveMessageRef はメッセージURLを解析し、同じサーバーか・実行者がメッセージを読めるかを確認する。
func (b *Bot) resolveMessageRef(ctx context.Context, e *events.ApplicationCommandInteractionCreate, rawURL string) (discordurl.MessageRef, string) {
	ref, err := discordurl.ParseMessageURL(rawURL)
	if err != nil {
		return ref, err.Error()
	}
	if ref.GuildID != *e.GuildID() {
		return ref, "違うサーバーの情報は操作できません。\nそのサーバーからコマンドを実行してください。"
	}
	ch, code := b.lookupChannel(ctx, ref.ChannelID)
	if code != "" {
		return ref, b.msgs.Display(code)
	}
	if ch.GuildID() != ref.GuildID {
		return ref, b.msgs.Display(i18n.ChannelNotFound)
	}
	perms := b.client.Caches.MemberPermissionsInChannel(ch, e.Member().Member)
	if codes := readPermCodes(perms); len(codes) > 0 {
		return ref, b.codesMessage(codes)
	}
	return ref, ""
}

// lookupChannel はキャッシュからチャンネルを探す。スレッドの場合は権限計算のため親チャンネルを返す。
func (b *Bot) lookupChannel(ctx context.Context, channelID snowflake.ID) (discord.GuildChannel, i18n.Code) {
	ch, ok := b.client.Caches.Channel(channelID)
	if !ok {
		// アーカイブ済みスレッドなどはキャッシュに無いので1度だけ取得する
		c, err := b.api.GetChannel(ctx, channelID)
		if err != nil {
			if discordapi.Classify(err) == discordapi.KindChannelDenied {
				return nil, i18n.ChannelAccessDenied
			}
			return nil, i18n.ChannelNotFound
		}
		gc, ok := c.(discord.GuildChannel)
		if !ok {
			return nil, i18n.ChannelNotFound
		}
		ch = gc
	}
	if th, ok := ch.(discord.GuildThread); ok {
		if pid := th.ParentID(); pid != nil {
			if parent, ok := b.client.Caches.Channel(*pid); ok {
				return parent, ""
			}
		}
		return nil, i18n.ChannelNotFound
	}
	return ch, ""
}

func (b *Bot) cmdExclusionAdd(e *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	if err := e.DeferCreateMessage(true); err != nil {
		b.log.Error("failed to defer", slog.Any("err", err))
		return
	}
	ctx := b.ctx
	ref, problem := b.resolveMessageRef(ctx, e, data.String("message_url"))
	if problem != "" {
		b.followup(e, discord.NewMessageUpdate().WithContent(problem))
		return
	}

	// 記録済みなら存在確認の API リクエストを省略する
	tracked, err := b.store.IsTracked(ctx, ref.MessageID)
	if err != nil {
		b.log.Error("failed to check tracked message", slog.Any("err", err))
	}
	if !tracked {
		if _, err := b.api.GetMessage(ctx, ref.ChannelID, ref.MessageID); err != nil {
			code := i18n.MessageNotFound
			if discordapi.Classify(err) == discordapi.KindChannelDenied {
				code = i18n.MessageAccessDenied
			}
			b.followup(e, discord.NewMessageUpdate().WithContent(b.msgs.Display(code)))
			return
		}
	}

	m := store.Message{ID: ref.MessageID, GuildID: ref.GuildID, ChannelID: ref.ChannelID}
	if err := b.store.AddExclusion(ctx, m); err != nil {
		b.log.Error("failed to add exclusion", slog.Any("err", err))
		b.followup(e, discord.NewMessageUpdate().WithContent(b.msgs.Display(i18n.Unknown)))
		return
	}
	b.followup(e, discord.NewMessageUpdate().WithContent(
		fmt.Sprintf("%sに除外設定を追加しました。", discord.MessageURL(ref.GuildID, ref.ChannelID, ref.MessageID))))
}

func (b *Bot) cmdExclusionRemove(e *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	ref, problem := b.resolveMessageRef(b.ctx, e, data.String("message_url"))
	if problem != "" {
		b.reply(e, problem)
		return
	}
	url := discord.MessageURL(ref.GuildID, ref.ChannelID, ref.MessageID)
	msg := fmt.Sprintf("%sの除外設定を解除しますか？\n除外設定解除によりメッセージが削除される場合があります。\n(続行する場合、%d秒以内に押してください)",
		url, int(b.cfg.ConfirmTimeout.Seconds()))
	b.confirms.ask(e, msg, func(ctx context.Context) string {
		existed, err := b.store.RemoveExclusion(ctx, store.Message{ID: ref.MessageID, GuildID: ref.GuildID, ChannelID: ref.ChannelID})
		if err != nil {
			b.log.Error("failed to remove exclusion", slog.Any("err", err))
			return b.msgs.Display(i18n.Unknown)
		}
		if !existed {
			return fmt.Sprintf("%sの除外設定はありません。", url)
		}
		b.deleter.Wake()
		return fmt.Sprintf("%sの除外設定を解除しました。", url)
	})
}

// ---------------------------------------------------------------------------
// /settings
// ---------------------------------------------------------------------------

const embedFieldLimit = 1024

func (b *Bot) cmdSettings(e *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	guildID := *e.GuildID()
	ch, hasChannel := data.OptChannel("channel")
	if hasChannel && !b.checkChannelPerms(e, ch, false) {
		return
	}
	ctx := b.ctx
	stats, err := b.store.ChannelStats(ctx, guildID)
	if err != nil {
		b.log.Error("failed to load stats", slog.Any("err", err))
		b.reply(e, b.msgs.Display(i18n.Unknown))
		return
	}
	var filter snowflake.ID
	if hasChannel {
		filter = ch.ID
	}
	exclusions, err := b.store.ListExclusions(ctx, guildID, filter)
	if err != nil {
		b.log.Error("failed to load exclusions", slog.Any("err", err))
		b.reply(e, b.msgs.Display(i18n.Unknown))
		return
	}

	var embed discord.Embed
	if hasChannel {
		entry, monitored := b.reg.Get(ch.ID)
		embed = discord.NewEmbed().WithTitle(fmt.Sprintf("%sの設定一覧", ch.Name))
		embed = embed.AddField("定期削除", map[bool]string{true: "有効", false: "無効"}[monitored], false)
		lt := "無効"
		if monitored {
			lt = lifetime.Format(entry.Lifetime, b.lang())
		}
		embed = embed.AddField("ライフタイム", lt, false)
		st := stats[ch.ID]
		embed = embed.AddField("削除予定", formatStat(st), false)
		var urls []string
		for _, x := range exclusions {
			urls = append(urls, discord.MessageURL(x.GuildID, x.ChannelID, x.ID))
		}
		embed = embed.AddField("除外したメッセージ", joinLimited(urls, "なし"), false)
	} else {
		guildName := "サーバー"
		if g, ok := b.client.Caches.Guild(guildID); ok {
			guildName = g.Name
		}
		embed = discord.NewEmbed().WithTitle(fmt.Sprintf("%sの設定一覧", guildName))
		var lines []string
		for _, entry := range sortedEntries(b.reg.Guild(guildID)) {
			line := fmt.Sprintf("%s: %s", mention(entry.ChannelID), lifetime.Format(entry.Lifetime, b.lang()))
			if st, ok := stats[entry.ChannelID]; ok {
				line += fmt.Sprintf("（%s）", formatStat(st))
			}
			lines = append(lines, line)
		}
		embed = embed.AddField("定期削除有効チャンネル: ライフタイム", joinLimited(lines, "なし"), false)

		var exLines []string
		var lastCh snowflake.ID
		for _, x := range exclusions {
			if x.ChannelID != lastCh {
				exLines = append(exLines, fmt.Sprintf("- %s:", mention(x.ChannelID)))
				lastCh = x.ChannelID
			}
			exLines = append(exLines, "  - "+discord.MessageURL(x.GuildID, x.ChannelID, x.ID))
		}
		embed = embed.AddField("除外したメッセージ", joinLimited(exLines, "なし"), false)
	}
	err = e.CreateMessage(discord.NewMessageCreate().WithEmbeds(embed).WithEphemeral(true).WithAllowedMentions(&discord.AllowedMentions{}))
	if err != nil {
		b.log.Error("failed to reply settings", slog.Any("err", err))
	}
}

func formatStat(st store.ChannelStat) string {
	if st.Pending == 0 || st.NextDelete == nil {
		return "削除予定なし"
	}
	return fmt.Sprintf("削除予定 %d件 / 次回 %s", st.Pending, discord.FormattedTimestampMention(st.NextDelete.Unix(), discord.TimestampStyleRelative))
}

// joinLimited は Embed フィールドの文字数上限に収まるよう行を連結する。
func joinLimited(lines []string, empty string) string {
	if len(lines) == 0 {
		return empty
	}
	var sb strings.Builder
	for i, l := range lines {
		suffix := fmt.Sprintf("\n…他%d件", len(lines)-i)
		if sb.Len()+len(l)+1+len(suffix) > embedFieldLimit {
			sb.WriteString(suffix)
			break
		}
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(l)
	}
	return sb.String()
}

func sortedEntries(es []registry.Entry) []registry.Entry {
	for i := 1; i < len(es); i++ {
		for j := i; j > 0 && es[j].ChannelID < es[j-1].ChannelID; j-- {
			es[j], es[j-1] = es[j-1], es[j]
		}
	}
	return es
}

// ---------------------------------------------------------------------------
// /clear
// ---------------------------------------------------------------------------

func (b *Bot) cmdClear(e *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	ch := data.Channel("channel")
	if !b.checkChannelPerms(e, ch, true) {
		return
	}
	var msg string
	if !b.reg.IsMonitored(ch.ID) {
		msg = fmt.Sprintf("%sは削除監視対象外のチャンネルです。\n続行する場合、除外したメッセージを除く全てのメッセージが削除されます。\n", mention(ch.ID))
	}
	msg += fmt.Sprintf("%sの削除を実行しますか？\n(続行する場合、%d秒以内に押してください)", mention(ch.ID), int(b.cfg.ConfirmTimeout.Seconds()))

	guildID := *e.GuildID()
	b.confirms.ask(e, msg, func(ctx context.Context) string {
		return b.clearChannel(ctx, guildID, ch.ID)
	})
}

// clearChannel はチャンネルのメッセージを即時削除予定として登録する。
// 実際の削除はスケジューラが一括削除 API でまとめて行うので、ここではメッセージ一覧の取得のみ行う。
func (b *Bot) clearChannel(ctx context.Context, guildID, channelID snowflake.ID) string {
	expedited, err := b.store.ExpediteChannel(ctx, channelID)
	if err != nil {
		b.log.Error("failed to expedite channel", slog.Any("err", err))
		return b.msgs.Display(i18n.Unknown)
	}

	var (
		before    snowflake.ID
		collected []store.Message
		limitHit  bool
	)
	for len(collected) < b.cfg.ClearMaxMessages {
		limit := min(100, b.cfg.ClearMaxMessages-len(collected))
		msgs, err := b.api.GetMessagesBefore(ctx, channelID, before, limit)
		if err != nil {
			b.log.Warn("failed to fetch messages for clear", slog.Uint64("channel_id", uint64(channelID)), slog.Any("err", err))
			if len(collected) == 0 && expedited == 0 {
				if discordapi.Classify(err) == discordapi.KindChannelDenied {
					return b.msgs.Display(i18n.MessageReadHistoryPermDenied)
				}
				return b.msgs.Display(i18n.MessageDeleteDenied)
			}
			break
		}
		for _, m := range msgs {
			if before == 0 || m.ID < before {
				before = m.ID
			}
			if m.Type.Deleteable() {
				collected = append(collected, store.Message{ID: m.ID, GuildID: guildID, ChannelID: channelID})
			}
		}
		if len(msgs) < limit {
			break
		}
		if len(collected) >= b.cfg.ClearMaxMessages {
			limitHit = true
		}
	}

	scheduled, err := b.store.ScheduleImmediate(ctx, collected)
	if err != nil {
		b.log.Error("failed to schedule clear", slog.Any("err", err))
		return b.msgs.Display(i18n.Unknown)
	}
	b.deleter.Wake()
	b.log.Info("channel clear scheduled", slog.Uint64("channel_id", uint64(channelID)),
		slog.Int("fetched", len(collected)), slog.Int64("scheduled", scheduled), slog.Int64("expedited", expedited))

	total := scheduled + expedited
	if total == 0 {
		return "削除対象がありませんでした。"
	}
	out := fmt.Sprintf("%sのメッセージ約%d件を削除予約しました。順次削除されます。", mention(channelID), max(scheduled, expedited))
	if limitHit {
		out += fmt.Sprintf("\n1度に予約できるのは%d件までです。\n削除完了後、続けて削除する場合はもう一度コマンドを実行してください。", b.cfg.ClearMaxMessages)
	}
	return out
}

// ---------------------------------------------------------------------------
// /help
// ---------------------------------------------------------------------------

func (b *Bot) cmdHelp(e *events.ApplicationCommandInteractionCreate) {
	m := b.msgs.Get
	desc, usage := m("HELP_DESCRIPTION_LABEL"), m("HELP_USAGE_LABEL")
	field := func(d, u string) string { return fmt.Sprintf("%s %s\n%s %s", desc, d, usage, u) }

	embed := discord.NewEmbed().
		WithTitle(m("HELP_TITLE")).
		WithDescription(m("HELP_DESCRIPTION")).
		WithColor(0x00ff00).
		AddField("`/enable` - "+m("HELP_COMMANDS_TITLE"), field(m("HELP_ENABLE_DESC"), "`/enable #general 1day`"), false).
		AddField("`/disable`", field(m("HELP_DISABLE_DESC"), "`/disable #general`"), false).
		AddField("`/exclusion_add`", field(m("HELP_EXCLUDE_ADD_DESC"), "`/exclusion_add https://discord.com/channels/...`"), false).
		AddField("`/exclusion_remove`", field(m("HELP_EXCLUDE_REMOVE_DESC"), "`/exclusion_remove https://discord.com/channels/...`"), false).
		AddField("`/settings`", fmt.Sprintf("%s %s\n%s `/settings` %s `/settings #general`", desc, m("HELP_SETTINGS_DESC"), usage, m("HELP_OR")), false).
		AddField("`/clear`", field(m("HELP_CLEAR_DESC"), "`/clear #general`"), false).
		AddField("", m("HELP_PERMISSION_NOTE")+"\n\n"+m("HELP_LIFETIME_EXAMPLES"), false).
		WithFooterText(m("HELP_FOOTER"))

	if err := e.CreateMessage(discord.NewMessageCreate().WithEmbeds(embed).WithEphemeral(true)); err != nil {
		b.log.Error("failed to reply help", slog.Any("err", err))
	}
}
