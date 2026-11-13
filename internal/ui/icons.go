package ui

import "fyne.io/fyne/v2"

// Clean filled-path SVG icons (24x24 viewBox) compatible with Fyne theme.NewThemedResource / NewColoredResource.
var (
	resIconPhone = fyne.NewStaticResource("phone.svg", []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M7.5 2A2.5 2.5 0 0 0 5 4.5v15A2.5 2.5 0 0 0 7.5 22h9a2.5 2.5 0 0 0 2.5-2.5v-15A2.5 2.5 0 0 0 16.5 2h-9zM7 4.5c0-.28.22-.5.5-.5h9c.28 0 .5.22.5.5v15c0 .28-.22.5-.5.5h-9a.5.5 0 0 1-.5-.5v-15zM10.25 5.25a.75.75 0 0 0 0 1.5h3.5a.75.75 0 0 0 0-1.5h-3.5zM11 17.5a1 1 0 1 0 2 0 1 1 0 0 0-2 0z"/></svg>`,
	))

	resIconDrive = fyne.NewStaticResource("drive.svg", []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M5.5 3A2.5 2.5 0 0 0 3 5.5v13A2.5 2.5 0 0 0 5.5 21h13a2.5 2.5 0 0 0 2.5-2.5v-13A2.5 2.5 0 0 0 18.5 3h-13zM5 5.5c0-.28.22-.5.5-.5h13c.28 0 .5.22.5.5V13H5V5.5zM5 15h14v3.5c0 .28-.22.5-.5.5h-13a.5.5 0 0 1-.5-.5V15zm11.5 1a1 1 0 1 0 0 2 1 1 0 0 0 0-2zm-3 0a1 1 0 1 0 0 2 1 1 0 0 0 0-2z"/></svg>`,
	))

	resIconSync = fyne.NewStaticResource("sync.svg", []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M12 4a8 8 0 0 0-6.71 3.65L3.8 6.16A.75.75 0 0 0 2.5 6.7V11c0 .41.34.75.75.75h4.3a.75.75 0 0 0 .53-1.28L6.68 9.07A6.01 6.01 0 0 1 17.85 10a.75.75 0 1 0 1.45-.38A8 8 0 0 0 12 4zm8.75 8.25h-4.3a.75.75 0 0 0-.53 1.28l1.4 1.4A6.01 6.01 0 0 1 6.15 14a.75.75 0 1 0-1.45.38A8 8 0 0 0 18.71 16.35l1.49 1.49a.75.75 0 0 0 1.3-.54V13a.75.75 0 0 0-.75-.75z"/></svg>`,
	))

	resIconRefresh = fyne.NewStaticResource("refresh.svg", []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M17.65 6.35A7.95 7.95 0 0 0 12 4a8 8 0 1 0 7.75 10h-2.1A6 6 0 1 1 12 6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/></svg>`,
	))

	resIconWifi = fyne.NewStaticResource("wifi.svg", []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M12 18a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3zm-4.95-2.45a.9.9 0 0 0 1.27 1.27 5.2 5.2 0 0 1 7.36 0 .9.9 0 1 0 1.27-1.27 7 7 0 0 0-9.9 0zm-3.18-3.18a.9.9 0 0 0 1.27 1.27 9.7 9.7 0 0 1 13.72 0 .9.9 0 0 0 1.27-1.27 11.5 11.5 0 0 0-16.26 0zM.7 9.18a.9.9 0 0 0 1.27 1.27 14.2 14.2 0 0 1 20.06 0 .9.9 0 1 0 1.27-1.27 16 16 0 0 0-22.6 0z"/></svg>`,
	))

	resIconFolder = fyne.NewStaticResource("folder.svg", []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M4.5 4A2.5 2.5 0 0 0 2 6.5v11A2.5 2.5 0 0 0 4.5 20h15a2.5 2.5 0 0 0 2.5-2.5V9a2.5 2.5 0 0 0-2.5-2.5h-7.17l-1.54-1.7A2.5 2.5 0 0 0 8.94 4H4.5zM4 6.5c0-.28.22-.5.5-.5h4.44c.14 0 .28.06.37.17L11.4 8.5h8.1c.28 0 .5.22.5.5v8.5c0 .28-.22.5-.5.5h-15a.5.5 0 0 1-.5-.5v-11z"/></svg>`,
	))

	resIconPhoto = fyne.NewStaticResource("photo.svg", []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M5 3a2.5 2.5 0 0 0-2.5 2.5v13A2.5 2.5 0 0 0 5 21h14a2.5 2.5 0 0 0 2.5-2.5v-13A2.5 2.5 0 0 0 19 3H5zm-.5 2.5c0-.28.22-.5.5-.5h14c.28 0 .5.22.5.5v9.09l-3.44-3.44a1.5 1.5 0 0 0-2.12 0l-3.69 3.69-1.69-1.69a1.5 1.5 0 0 0-2.12 0L4.5 15.09V5.5zM8.5 7.5a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3z"/></svg>`,
	))

	resIconVideo = fyne.NewStaticResource("video.svg", []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M4.5 5A2.5 2.5 0 0 0 2 7.5v9A2.5 2.5 0 0 0 4.5 19h10a2.5 2.5 0 0 0 2.5-2.5v-1.76l3.36 2.1A1.1 1.1 0 0 0 22 15.9V8.1a1.1 1.1 0 0 0-1.64-.94L17 9.26V7.5A2.5 2.5 0 0 0 14.5 5h-10zM4 7.5c0-.28.22-.5.5-.5h10c.28 0 .5.22.5.5v9c0 .28-.22.5-.5.5h-10a.5.5 0 0 1-.5-.5v-9z"/></svg>`,
	))

	resIconSettings = fyne.NewStaticResource("settings.svg", []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M19.14 12.94c.04-.3.06-.61.06-.94 0-.32-.02-.64-.07-.94l2.03-1.58a.49.49 0 0 0 .12-.61l-1.92-3.32a.49.49 0 0 0-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54a.48.48 0 0 0-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96a.49.49 0 0 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.05.3-.09.63-.09.94s.02.64.07.94l-2.03 1.58a.49.49 0 0 0-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6A3.6 3.6 0 1 1 12 8.4a3.6 3.6 0 0 1 0 7.2z"/></svg>`,
	))

	resIconSearch = fyne.NewStaticResource("search.svg", []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M10.5 3a7.5 7.5 0 1 0 4.65 13.38l4.23 4.24a1 1 0 0 0 1.42-1.42l-4.24-4.23A7.5 7.5 0 0 0 10.5 3zm-5.5 7.5a5.5 5.5 0 1 1 11 0 5.5 5.5 0 0 1-11 0z"/></svg>`,
	))

	resIconStop = fyne.NewStaticResource("stop.svg", []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M6.5 5A1.5 1.5 0 0 0 5 6.5v11A1.5 1.5 0 0 0 6.5 19h11a1.5 1.5 0 0 0 1.5-1.5v-11A1.5 1.5 0 0 0 17.5 5h-11z"/></svg>`,
	))

	resIconCheck = fyne.NewStaticResource("check.svg", []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M19.7 6.3a1 1 0 0 1 0 1.4l-9.5 9.5a1 1 0 0 1-1.4 0l-4.5-4.5a1 1 0 1 1 1.4-1.4l3.8 3.79 8.8-8.79a1 1 0 0 1 1.4 0z"/></svg>`,
	))

	resIconChevron = fyne.NewStaticResource("chevron.svg", []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M6.3 9.3a1 1 0 0 1 1.4 0L12 13.58l4.3-4.29a1 1 0 1 1 1.4 1.42l-5 5a1 1 0 0 1-1.4 0l-5-5a1 1 0 0 1 0-1.42z"/></svg>`,
	))

	resIconLog = fyne.NewStaticResource("log.svg", []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M5 6a1 1 0 0 1 1-1h12a1 1 0 1 1 0 2H6a1 1 0 0 1-1-1zm0 6a1 1 0 0 1 1-1h12a1 1 0 1 1 0 2H6a1 1 0 0 1-1-1zm0 6a1 1 0 0 1 1-1h8a1 1 0 1 1 0 2H6a1 1 0 0 1-1-1z"/></svg>`,
	))

	resIconAppLogo = fyne.NewStaticResource("app_logo.svg", []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M9.4 4.2a2 2 0 0 0-1.66.89L6.6 6.8H4.5A2.5 2.5 0 0 0 2 9.3v8.2A2.5 2.5 0 0 0 4.5 20h15a2.5 2.5 0 0 0 2.5-2.5V9.3a2.5 2.5 0 0 0-2.5-2.5h-2.1l-1.14-1.71a2 2 0 0 0-1.66-.89H9.4zm2.6 5.05a4.25 4.25 0 1 1 0 8.5 4.25 4.25 0 0 1 0-8.5zm0 2a2.25 2.25 0 1 0 0 4.5 2.25 2.25 0 0 0 0-4.5z"/></svg>`,
	))
)
