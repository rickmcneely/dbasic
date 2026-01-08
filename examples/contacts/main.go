package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

// ContactModel implements walk.TableModel for the contact list
type ContactModel struct {
	walk.TableModelBase
	walk.SorterBase
	sortColumn int
	sortOrder  walk.SortOrder
	items      []*Contact
	db         *sql.DB
}

func NewContactModel(db *sql.DB) *ContactModel {
	m := &ContactModel{
		db:         db,
		sortColumn: 1, // Default sort by last name
		sortOrder:  walk.SortAscending,
	}
	m.ResetRows()
	return m
}

func (m *ContactModel) RowCount() int {
	return len(m.items)
}

func (m *ContactModel) Value(row, col int) interface{} {
	if row < 0 || row >= len(m.items) {
		return nil
	}
	item := m.items[row]
	switch col {
	case 0:
		return item.FirstName
	case 1:
		return item.LastName
	case 2:
		return item.City
	case 3:
		return item.State
	case 4:
		return item.Phone
	}
	return nil
}

func (m *ContactModel) Sort(col int, order walk.SortOrder) error {
	m.sortColumn = col
	m.sortOrder = order

	sort.SliceStable(m.items, func(i, j int) bool {
		a, b := m.items[i], m.items[j]
		var less bool
		switch col {
		case 0:
			less = a.FirstName < b.FirstName
		case 1:
			less = a.LastName < b.LastName
		case 2:
			less = a.City < b.City
		case 3:
			less = a.State < b.State
		case 4:
			less = a.Phone < b.Phone
		}
		if order == walk.SortDescending {
			return !less
		}
		return less
	})

	m.PublishRowsReset()
	return nil
}

func (m *ContactModel) ResetRows() {
	var sortBy string
	switch m.sortColumn {
	case 0:
		sortBy = "first_name"
	case 1:
		sortBy = "last_name"
	case 2:
		sortBy = "city"
	case 3:
		sortBy = "state"
	case 4:
		sortBy = "phone"
	}
	sortAsc := m.sortOrder == walk.SortAscending

	contacts, err := GetAllContacts(m.db, sortBy, sortAsc)
	if err != nil {
		log.Printf("Error loading contacts: %v", err)
		return
	}
	m.items = contacts
	m.PublishRowsReset()
}

func (m *ContactModel) Search(query string) {
	var sortBy string
	switch m.sortColumn {
	case 0:
		sortBy = "first_name"
	case 1:
		sortBy = "last_name"
	case 2:
		sortBy = "city"
	case 3:
		sortBy = "state"
	case 4:
		sortBy = "phone"
	}
	sortAsc := m.sortOrder == walk.SortAscending

	if query == "" {
		m.ResetRows()
		return
	}

	contacts, err := SearchContacts(m.db, query, sortBy, sortAsc)
	if err != nil {
		log.Printf("Error searching contacts: %v", err)
		return
	}
	m.items = contacts
	m.PublishRowsReset()
}

func (m *ContactModel) Item(index int) *Contact {
	if index < 0 || index >= len(m.items) {
		return nil
	}
	return m.items[index]
}

// Application holds the main window components
type Application struct {
	mainWindow   *walk.MainWindow
	tableView    *walk.TableView
	model        *ContactModel
	statusBar    *walk.StatusBarItem
	searchEdit   *walk.LineEdit
	db           *sql.DB
}

// showError displays an error message box (works in GUI mode)
func showError(title, message string) {
	walk.MsgBox(nil, title, message, walk.MsgBoxIconError)
}

func main() {
	// Catch any panics and display them
	defer func() {
		if r := recover(); r != nil {
			showError("Panic", fmt.Sprintf("Application crashed: %v", r))
		}
	}()
	// Get database path in same directory as executable
	exePath, err := os.Executable()
	if err != nil {
		showError("Startup Error", fmt.Sprintf("Failed to get executable path: %v", err))
		return
	}
	dbPath := filepath.Join(filepath.Dir(exePath), "contacts.db")

	// Initialize database
	db, err := InitDatabase(dbPath)
	if err != nil {
		showError("Database Error", fmt.Sprintf("Failed to initialize database: %v\n\nPath: %s", err, dbPath))
		return
	}
	defer db.Close()

	// Create tables and seed data
	if err := CreateContactsTable(db); err != nil {
		showError("Database Error", fmt.Sprintf("Failed to create tables: %v", err))
		return
	}
	if err := SeedDatabase(db); err != nil {
		showError("Database Error", fmt.Sprintf("Failed to seed database: %v", err))
		return
	}

	// Create application
	app := &Application{db: db}
	app.model = NewContactModel(db)

	// Run the main window
	if err := app.Run(); err != nil {
		showError("Application Error", fmt.Sprintf("Failed to run application: %v", err))
		return
	}
}

func (app *Application) Run() error {
	var searchEdit *walk.LineEdit

	_, err := MainWindow{
		AssignTo: &app.mainWindow,
		Title:    "Contact Book",
		MinSize:  Size{Width: 800, Height: 600},
		Size:     Size{Width: 900, Height: 650},
		Layout:   VBox{MarginsZero: true},
		MenuItems: []MenuItem{
			Menu{
				Text: "&File",
				Items: []MenuItem{
					Action{
						Text:        "&New Contact\tCtrl+N",
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyN},
						OnTriggered: app.onNewContact,
					},
					Separator{},
					Action{
						Text:        "E&xit\tAlt+F4",
						OnTriggered: func() { app.mainWindow.Close() },
					},
				},
			},
			Menu{
				Text: "&Edit",
				Items: []MenuItem{
					Action{
						Text:        "&Edit Contact\tEnter",
						Shortcut:    Shortcut{Key: walk.KeyReturn},
						OnTriggered: app.onEditContact,
					},
					Action{
						Text:        "&Delete Contact\tDel",
						Shortcut:    Shortcut{Key: walk.KeyDelete},
						OnTriggered: app.onDeleteContact,
					},
				},
			},
			Menu{
				Text: "&View",
				Items: []MenuItem{
					Action{
						Text:        "&Refresh\tF5",
						Shortcut:    Shortcut{Key: walk.KeyF5},
						OnTriggered: app.onRefresh,
					},
				},
			},
			Menu{
				Text: "&Help",
				Items: []MenuItem{
					Action{
						Text:        "&About",
						OnTriggered: app.onAbout,
					},
				},
			},
		},
		ToolBar: ToolBar{
			ButtonStyle: ToolBarButtonImageBeforeText,
			Items: []MenuItem{
				Action{
					Text:        "New",
					OnTriggered: app.onNewContact,
				},
				Action{
					Text:        "Edit",
					OnTriggered: app.onEditContact,
				},
				Action{
					Text:        "Delete",
					OnTriggered: app.onDeleteContact,
				},
				Separator{},
				Action{
					Text:        "Refresh",
					OnTriggered: app.onRefresh,
				},
			},
		},
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					Label{Text: "Search:"},
					LineEdit{
						AssignTo:      &searchEdit,
						MaxSize:       Size{Width: 200},
						OnTextChanged: func() { app.model.Search(searchEdit.Text()) },
					},
				},
			},
			TableView{
				AssignTo:         &app.tableView,
				AlternatingRowBG: true,
				ColumnsOrderable: true,
				MultiSelection:   false,
				Model:            app.model,
				Columns: []TableViewColumn{
					{Title: "First Name", Width: 120},
					{Title: "Last Name", Width: 120},
					{Title: "City", Width: 150},
					{Title: "State", Width: 60},
					{Title: "Phone", Width: 120},
				},
				OnItemActivated: app.onEditContact,
			},
		},
		StatusBarItems: []StatusBarItem{
			{
				AssignTo: &app.statusBar,
				Width:    200,
			},
		},
	}.Run()

	return err
}

func (app *Application) updateStatus() {
	count := len(app.model.items)
	total := ContactCount(app.db)
	if count == total {
		app.statusBar.SetText(fmt.Sprintf("%d contacts", count))
	} else {
		app.statusBar.SetText(fmt.Sprintf("%d of %d contacts", count, total))
	}
}

func (app *Application) onNewContact() {
	contact := &Contact{}
	if app.showContactDialog(contact, "New Contact") {
		if err := InsertContact(app.db, contact); err != nil {
			walk.MsgBox(app.mainWindow, "Error", fmt.Sprintf("Failed to create contact: %v", err), walk.MsgBoxIconError)
			return
		}
		app.model.ResetRows()
		app.updateStatus()
	}
}

func (app *Application) onEditContact() {
	index := app.tableView.CurrentIndex()
	if index < 0 {
		walk.MsgBox(app.mainWindow, "No Selection", "Please select a contact to edit.", walk.MsgBoxIconInformation)
		return
	}

	original := app.model.Item(index)
	if original == nil {
		return
	}

	// Create a copy for editing
	contact := &Contact{
		ID:        original.ID,
		FirstName: original.FirstName,
		LastName:  original.LastName,
		Address:   original.Address,
		City:      original.City,
		State:     original.State,
		Zip:       original.Zip,
		Phone:     original.Phone,
		Email:     original.Email,
	}

	if app.showContactDialog(contact, "Edit Contact") {
		if err := UpdateContact(app.db, contact); err != nil {
			walk.MsgBox(app.mainWindow, "Error", fmt.Sprintf("Failed to update contact: %v", err), walk.MsgBoxIconError)
			return
		}
		app.model.ResetRows()
	}
}

func (app *Application) onDeleteContact() {
	index := app.tableView.CurrentIndex()
	if index < 0 {
		walk.MsgBox(app.mainWindow, "No Selection", "Please select a contact to delete.", walk.MsgBoxIconInformation)
		return
	}

	contact := app.model.Item(index)
	if contact == nil {
		return
	}

	result := walk.MsgBox(
		app.mainWindow,
		"Confirm Delete",
		fmt.Sprintf("Are you sure you want to delete %s %s?", contact.FirstName, contact.LastName),
		walk.MsgBoxYesNo|walk.MsgBoxIconQuestion,
	)

	if result == walk.DlgCmdYes {
		if err := DeleteContact(app.db, contact.ID); err != nil {
			walk.MsgBox(app.mainWindow, "Error", fmt.Sprintf("Failed to delete contact: %v", err), walk.MsgBoxIconError)
			return
		}
		app.model.ResetRows()
		app.updateStatus()
	}
}

func (app *Application) onRefresh() {
	app.model.ResetRows()
	app.updateStatus()
}

func (app *Application) onAbout() {
	walk.MsgBox(
		app.mainWindow,
		"About Contact Book",
		"Contact Book v1.0\n\nA Win32 SQLite contact manager.\nBuilt with Go and lxn/walk.\n\nDBasic Example - Go Package Integration",
		walk.MsgBoxIconInformation,
	)
}

func (app *Application) showContactDialog(contact *Contact, title string) bool {
	var dlg *walk.Dialog
	var firstNameEdit, lastNameEdit, addressEdit, cityEdit, stateEdit, zipEdit, phoneEdit, emailEdit *walk.LineEdit
	var acceptPB, cancelPB *walk.PushButton

	Dialog{
		AssignTo:      &dlg,
		Title:         title,
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		MinSize:       Size{Width: 400, Height: 350},
		Layout:        VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{Text: "First Name:"},
					LineEdit{AssignTo: &firstNameEdit, Text: contact.FirstName},

					Label{Text: "Last Name:"},
					LineEdit{AssignTo: &lastNameEdit, Text: contact.LastName},

					Label{Text: "Address:"},
					LineEdit{AssignTo: &addressEdit, Text: contact.Address},

					Label{Text: "City:"},
					LineEdit{AssignTo: &cityEdit, Text: contact.City},

					Label{Text: "State:"},
					LineEdit{AssignTo: &stateEdit, Text: contact.State, MaxLength: 2},

					Label{Text: "Zip:"},
					LineEdit{AssignTo: &zipEdit, Text: contact.Zip, MaxLength: 10},

					Label{Text: "Phone:"},
					LineEdit{AssignTo: &phoneEdit, Text: contact.Phone},

					Label{Text: "Email:"},
					LineEdit{AssignTo: &emailEdit, Text: contact.Email},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &acceptPB,
						Text:     "OK",
						OnClicked: func() {
							contact.FirstName = firstNameEdit.Text()
							contact.LastName = lastNameEdit.Text()
							contact.Address = addressEdit.Text()
							contact.City = cityEdit.Text()
							contact.State = stateEdit.Text()
							contact.Zip = zipEdit.Text()
							contact.Phone = phoneEdit.Text()
							contact.Email = emailEdit.Text()

							// Validate required fields
							if contact.FirstName == "" || contact.LastName == "" {
								walk.MsgBox(dlg, "Validation Error", "First Name and Last Name are required.", walk.MsgBoxIconWarning)
								return
							}

							dlg.Accept()
						},
					},
					PushButton{
						AssignTo:  &cancelPB,
						Text:      "Cancel",
						OnClicked: func() { dlg.Cancel() },
					},
				},
			},
		},
	}.Run(app.mainWindow)

	return dlg.Result() == walk.DlgCmdOK
}
