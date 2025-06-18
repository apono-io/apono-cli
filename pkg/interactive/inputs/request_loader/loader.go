package requestloader

import (
	"context"
	"fmt"
	"time"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/services"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gookit/color"
)

const (
	abortingText = "Aborting..."
	interval     = 1 * time.Second
)

func (m model) Init() tea.Cmd {
	return tea.Batch(getRequestByID(m.ctx, m.client, m.requestID), m.spinner.Tick)
}

func (e errMsg) Error() string { return e.err.Error() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updatedRequestMsg:
		m.request = (*clientapi.AccessRequestClientModel)(&msg)
		if m.noWaitForGrant || ShouldStopLoading(m.request) {
			m.quitting = true
			return m, tea.Quit
		}
		if time.Now().After(m.startLoadingTime.Add(m.timeout)) {
			m.err = fmt.Errorf("timeout waiting for request to be granted")
			return m, tea.Quit
		}
		if shouldRetryLoading(m.lastRequestTime, interval) {
			m.lastRequestTime = time.Now()
			return m, getRequestByID(m.ctx, m.client, m.request.Id)
		}

		return m, func() tea.Msg { return updatedRequestMsg(*m.request) }

	case errMsg:
		m.err = msg
		return m, tea.Quit

	case tea.KeyMsg:
		if msg.String() == abortKey {
			m.aborting = true
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	if m.aborting {
		return abortingText
	}

	var msg string
	if m.request == nil {
		msg = fmt.Sprintf("%s Waiting for request to be ready", m.spinner.View())
	} else {
		msg = fmt.Sprintf("%s Request %s is %s", m.spinner.View(), color.Bold.Sprint(m.request.Id), services.ColoredStatus(*m.request))
	}

	return "\n" + msg + "\n\n"
}

func RunRequestLoader(ctx context.Context, client *aponoapi.AponoClient, requestID string, timeout time.Duration, noWaitForGrant bool) (*clientapi.AccessRequestClientModel, error) {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	initModel := model{
		spinner:          s,
		ctx:              ctx,
		client:           client,
		requestID:        requestID,
		timeout:          timeout,
		startLoadingTime: time.Now(),
		lastRequestTime:  time.Now(),
		noWaitForGrant:   noWaitForGrant,
	}

	result, err := tea.NewProgram(initModel).Run()
	if err != nil {
		return nil, err
	}

	resultModel := result.(model)
	if resultModel.err != nil {
		return nil, resultModel.err
	}

	if resultModel.aborting {
		return nil, fmt.Errorf("aborted by user")
	}

	return resultModel.request, nil
}
