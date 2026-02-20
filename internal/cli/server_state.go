package cli

import (
	"github.com/Zyrakk/zplay/internal/config"
	"github.com/Zyrakk/zplay/internal/games"
)

func resolveNamespace(srv config.ServerInfo, game games.Game) string {
	if srv.Namespace != "" {
		return srv.Namespace
	}
	return game.GetNamespace(srv.Name)
}
