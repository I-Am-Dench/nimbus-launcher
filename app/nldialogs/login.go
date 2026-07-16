package nldialogs

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/origin"
)

func AskForCredentials(ctx origin.AuthContext) (username string, password []byte, err error) {
	if len(ctx.Message) > 0 {
		slog.Error("Failed to authenticate", "message", ctx.Message)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	usernameEntry := widget.NewEntry()
	usernameEntry.PlaceHolder = "nexus"

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.PlaceHolder = "password123"

	submitted := false

	fyne.Do(func() {
		window := fyne.CurrentApp().NewWindow("Authenticate")
		window.SetIcon(theme.AccountIcon())
		window.SetOnClosed(func() {
			wg.Done()
		})
		window.Resize(fyne.NewSize(400, 200))
		window.SetFixedSize(true)

		heading := canvas.NewText("Authenticate", theme.Color(theme.ColorNameForeground))
		heading.TextSize = 16

		url := widget.NewLabel(fmt.Sprint(ctx.URL.Scheme, "://", ctx.URL.Host))

		submit := func(_ string) {
			if len(usernameEntry.Text) > 0 && len(passwordEntry.Text) > 0 {
				submitted = true
				window.Close()
			}
		}

		submitButton := widget.NewButtonWithIcon("Submit", theme.ConfirmIcon(), func() {
			submitted = true
			window.Close()
		})
		submitButton.Disable()
		submitButton.Importance = widget.HighImportance

		cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), window.Close)

		usernameEntry.OnChanged = func(s string) {
			if len(usernameEntry.Text) == 0 || len(passwordEntry.Text) == 0 {
				submitButton.Disable()
			} else {
				submitButton.Enable()
			}
		}
		passwordEntry.OnChanged = usernameEntry.OnChanged

		usernameEntry.OnSubmitted = submit
		passwordEntry.OnSubmitted = submit

		buttons := container.NewBorder(nil, nil, nil, container.NewHBox(cancelButton, submitButton))
		buttons.Resize(fyne.NewSize(400, 200))

		errorMessage := widget.NewLabel(ctx.Message)
		errorMessage.Importance = widget.DangerImportance
		errorMessage.Wrapping = fyne.TextWrapWord

		window.SetContent(container.NewPadded(
			container.NewBorder(
				container.NewVBox(heading, url), buttons, nil, nil, container.NewPadded(
					container.NewVBox(
						widget.NewForm(
							widget.NewFormItem("Username", usernameEntry),
							widget.NewFormItem("Password", passwordEntry),
						),
						errorMessage,
					),
				),
			),
		))

		window.Canvas().Focus(usernameEntry)

		window.CenterOnScreen()
		window.Show()
	})

	wg.Wait()
	if !submitted {
		err = context.Canceled
	}

	return usernameEntry.Text, []byte(passwordEntry.Text), err
}
