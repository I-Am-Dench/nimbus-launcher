package app

import (
	"log"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func HyperLinkButton(text string, icon fyne.Resource, urlBinding binding.String) *widget.Button {
	button := widget.NewButtonWithIcon(text, icon,
		func() {
			rawURL, _ := urlBinding.Get()
			if len(rawURL) == 0 {
				return
			}

			url, err := url.Parse(rawURL)
			if err != nil {
				log.Printf("cannot parse URL \"%s\": %v", rawURL, err)
				return
			}

			log.Printf("Opening link: %v\n", url)
			err = fyne.CurrentApp().OpenURL(url)
			if err != nil {
				log.Printf("could not open URL \"%s\": %v\n", url, err)
			}
		})

	button.Importance = widget.LowImportance
	button.Alignment = widget.ButtonAlignLeading

	return button
}

func AddEllipsis(label *widget.Label) *widget.Label {
	label.Truncation = fyne.TextTruncateEllipsis
	return label
}

func BackButton(tapped func()) *widget.Button {
	button := widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), tapped)
	button.Importance = widget.LowImportance

	return button
}
