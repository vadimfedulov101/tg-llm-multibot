package names

type Names struct {
	Bot  string
	User string
}

func New(bot string, user string) *Names {
	return &Names{
		Bot:  bot,
		User: user,
	}
}
