package emoji

import (
	"github.com/deta/pc-cli/pkg/components/styles"
)

var (
	Cowboy           = Emoji{Emoji: "🤠 ", Fallback: ""}
	Gear             = Emoji{Emoji: "⚙️ ", Fallback: ""}
	PointDown        = Emoji{Emoji: "👇 ", Fallback: ""}
	Link             = Emoji{Emoji: "🔗 ", Fallback: ""}
	ErrorExclamation = Emoji{Emoji: "❗", Fallback: styles.ErrorExclamation}
	ThumbsUp         = Emoji{Emoji: "👍 ", Fallback: styles.CheckMark}
	Check            = Emoji{Emoji: styles.CheckMark, Fallback: styles.CheckMark}
	PartyPopper      = Emoji{Emoji: "🎉 ", Fallback: styles.CheckMark}
	Rocket           = Emoji{Emoji: "🚀 ", Fallback: ""}
	Earth            = Emoji{Emoji: "🌍 ", Fallback: ""}
	PartyFace        = Emoji{Emoji: "🥳 ", Fallback: ""}
	X                = Emoji{Emoji: "❌ ", Fallback: styles.X}
	Waving           = Emoji{Emoji: "👋 ", Fallback: ""}
	Swirl            = Emoji{Emoji: "🌀 ", Fallback: ""}
	Sparkles         = Emoji{Emoji: "✨ ", Fallback: styles.CheckMark}
	Files            = Emoji{Emoji: "🗂️ ", Fallback: ""}
	Package          = Emoji{Emoji: "📦 ", Fallback: styles.Boldf("~")}
	Eyes             = Emoji{Emoji: "👀 ", Fallback: ""}
	Lightning        = Emoji{Emoji: "⚡ ", Fallback: ""}
	Pistol           = Emoji{Emoji: "🔫 ", Fallback: ""}
	Tools            = Emoji{Emoji: "💻 ", Fallback: styles.Info}
	CrystalBall      = Emoji{Emoji: "🔮 ", Fallback: ""}
	Label            = Emoji{Emoji: "🏷️ ", Fallback: ""}
	Key              = Emoji{Emoji: "🔑 ", Fallback: ""}
)
