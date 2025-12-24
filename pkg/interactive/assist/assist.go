package assist

import (
	"context"

	"github.com/apono-io/apono-cli/pkg/aponoapi"

	tea "github.com/charmbracelet/bubbletea"
)

func RunAssistant(ctx context.Context, client *aponoapi.AponoClient) error {
	m := NewModel(ctx, client)

	p := tea.NewProgram(m, tea.WithContext(ctx))

	_, err := p.Run()
	return err
}
