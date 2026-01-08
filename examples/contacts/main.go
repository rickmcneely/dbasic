package main

import (
	"github.com/lxn/walk/declarative"
	_ "modernc.org/sqlite"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"github.com/lxn/walk"
)

// Runtime helper functions

// Str converts a number to string
func Str(val interface{}) string {
	return fmt.Sprintf("%v", val)
}

type Contact struct {
	ID int64
	FirstName string
	LastName string
	Address string
	City string
	State string
	Zip string
	Phone string
	Email string
}

type ContactModel struct {
	walk.TableModelBase
	walk.SorterBase
	SortCol int
	SortAsc bool
	Items []*Contact
	DB *sql.DB
	TV *walk.TableView
}

type DialogEdits struct {
	firstName *walk.LineEdit
	lastName *walk.LineEdit
	address *walk.LineEdit
	city *walk.LineEdit
	state *walk.LineEdit
	zip *walk.LineEdit
	phone *walk.LineEdit
	email *walk.LineEdit
	contact *Contact
	dialog *walk.Dialog
}


var (
	gMainWindow *walk.MainWindow
	gModel *ContactModel
	gStatusBar *walk.StatusBarItem
	gDialogEdits DialogEdits
)


func (m *ContactModel) RowCount() int {
	return len((*m).Items)
}

func (m *ContactModel) Value(row int, col int) interface{} {
	if ((row < 0) || (row >= len((*m).Items))) {
		return nil
	}
	var item *Contact = (*m).Items[row]
	switch col {
	case 0:
		return (*item).FirstName
	case 1:
		return (*item).LastName
	case 2:
		return (*item).City
	case 3:
		return (*item).State
	case 4:
		return (*item).Phone
	}
	return nil
}

func (m *ContactModel) Sort(col int, order walk.SortOrder) error {
	(*m).SortCol = col
	(*m).SortAsc = (order == walk.SortAscending)
	(*m).Items = GetAllContacts((*m).DB, col, (*m).SortAsc)
	(*m).PublishRowsReset()
	if ((*m).TV != nil) {
		(*(*m).TV).Invalidate()
	}
	return nil
}

func (m *ContactModel) ResetRows()  {
	(*m).Items = GetAllContacts((*m).DB, (*m).SortCol, (*m).SortAsc)
	(*m).PublishRowsReset()
}

func (m *ContactModel) Item(index int) *Contact {
	if ((index < 0) || (index >= len((*m).Items))) {
		return nil
	}
	return (*m).Items[index]
}

func InitDatabase(dbPath string) *sql.DB {
	var db *sql.DB
	var err error
	db, err = sql.Open("sqlite", dbPath)
	if (err != nil) {
		return nil
	}
	return db
}

func CreateContactsTable(db *sql.DB) {
	var query string = (((("CREATE TABLE IF NOT EXISTS contacts (" + "id INTEGER PRIMARY KEY AUTOINCREMENT, ") + "first_name TEXT NOT NULL, ") + "last_name TEXT NOT NULL, ") + "address TEXT, city TEXT, state TEXT, zip TEXT, phone TEXT, email TEXT)")
	(*db).Exec(query)
}

func GetAllContacts(db *sql.DB, sortCol int, sortAsc bool) []*Contact {
	var contacts []*Contact
	var sortBy string
	var orderDir string
	switch sortCol {
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
	default:
		sortBy = "last_name"
	}
	if sortAsc {
		orderDir = "ASC"
	} else {
		orderDir = "DESC"
	}
	var query string = ((("SELECT id, first_name, last_name, address, city, state, zip, phone, email FROM contacts ORDER BY " + sortBy) + " ") + orderDir)
	var rows *sql.Rows
	var err error
	rows, err = (*db).Query(query)
	if (err != nil) {
		return contacts
	}
	for (*rows).Next() {
		var c *Contact = new(Contact)
		(*rows).Scan(&(*c).ID, &(*c).FirstName, &(*c).LastName, &(*c).Address, &(*c).City, &(*c).State, &(*c).Zip, &(*c).Phone, &(*c).Email)
		contacts = append(contacts, c)
	}
	(*rows).Close()
	return contacts
}

func InsertContact(db *sql.DB, c *Contact) {
	var query string = "INSERT INTO contacts (first_name, last_name, address, city, state, zip, phone, email) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
	(*db).Exec(query, (*c).FirstName, (*c).LastName, (*c).Address, (*c).City, (*c).State, (*c).Zip, (*c).Phone, (*c).Email)
}

func UpdateContact(db *sql.DB, c *Contact) {
	var query string = "UPDATE contacts SET first_name=?, last_name=?, address=?, city=?, state=?, zip=?, phone=?, email=? WHERE id=?"
	(*db).Exec(query, (*c).FirstName, (*c).LastName, (*c).Address, (*c).City, (*c).State, (*c).Zip, (*c).Phone, (*c).Email, (*c).ID)
}

func DeleteContact(db *sql.DB, id int64) {
	(*db).Exec("DELETE FROM contacts WHERE id=?", id)
}

func SeedDatabase(db *sql.DB) {
	var count int
	(*db).QueryRow("SELECT COUNT(*) FROM contacts").Scan(&count)
	if (count > 0) {
		return
	}
	var seedContacts []Contact = []Contact{Contact{Email: "herman@munster.example", FirstName: "Herman", LastName: "Munster", Address: "1313 Mockingbird Lane", City: "Mockingbird Heights", State: "CA", Zip: "90210", Phone: "555-555-0001"}, Contact{Zip: "07090", Phone: "555-555-0002", Email: "gomez@addams.example", FirstName: "Gomez", LastName: "Addams", Address: "0001 Cemetery Lane", City: "Westfield", State: "NJ"}, Contact{Zip: "49007", Phone: "555-555-0005", Email: "homer@simpson.example", FirstName: "Homer", LastName: "Simpson", Address: "742 Evergreen Terrace", City: "Springfield", State: "OR"}, Contact{FirstName: "Al", LastName: "Bundy", Address: "9764 Jeopardy Lane", City: "Chicago", State: "IL", Zip: "60614", Phone: "555-555-0007", Email: "al@bundy.example"}, Contact{Phone: "555-555-0022", Email: "walter@white.example", FirstName: "Walter", LastName: "White", Address: "308 Negra Arroyo Lane", City: "Albuquerque", State: "NM", Zip: "87104"}, Contact{Address: "1725 Slough Avenue", City: "Scranton", State: "PA", Zip: "18503", Phone: "555-555-0024", Email: "michael@scott.example", FirstName: "Michael", LastName: "Scott"}}
	var i int
	for i = 0; i <= (len(seedContacts) - 1); i += 1 {
		var c *Contact = &seedContacts[i]
		InsertContact(db, c)
	}
}

func OnNewContact() {
	var c Contact
	if ShowContactDialog(&c, "New Contact") {
		InsertContact((*gModel).DB, &c)
		(*gModel).ResetRows()
		UpdateStatus()
	}
}

func OnEditContact() {
	var tv *walk.TableView = (*gModel).TV
	if (tv == nil) {
		return
	}
	var index int = (*tv).CurrentIndex()
	if (index < 0) {
		walk.MsgBox(gMainWindow, "No Selection", "Please select a contact to edit.", walk.MsgBoxIconInformation)
		return
	}
	var c *Contact = (*gModel).Item(index)
	if (c == nil) {
		return
	}
	var editCopy Contact = (*c)
	if ShowContactDialog(&editCopy, "Edit Contact") {
		UpdateContact((*gModel).DB, &editCopy)
		(*gModel).ResetRows()
	}
}

func OnDeleteContact() {
	var tv *walk.TableView = (*gModel).TV
	if (tv == nil) {
		return
	}
	var index int = (*tv).CurrentIndex()
	if (index < 0) {
		walk.MsgBox(gMainWindow, "No Selection", "Please select a contact to delete.", walk.MsgBoxIconInformation)
		return
	}
	var c *Contact = (*gModel).Item(index)
	if (c == nil) {
		return
	}
	var result int
	result = walk.MsgBox(gMainWindow, "Confirm Delete", (((("Delete " + (*c).FirstName) + " ") + (*c).LastName) + "?"), (walk.MsgBoxYesNo + walk.MsgBoxIconQuestion))
	if (result == walk.DlgCmdYes) {
		DeleteContact((*gModel).DB, (*c).ID)
		(*gModel).ResetRows()
		UpdateStatus()
	}
}

func UpdateStatus() {
	var count int = (*gModel).RowCount()
	(*gStatusBar).SetText((Str(count) + " contacts"))
}

func ShowContactDialog(c *Contact, title string) bool {
	var dlg *walk.Dialog
	var acceptPB *walk.PushButton
	var cancelPB *walk.PushButton
	var firstNameEdit *walk.LineEdit
	var lastNameEdit *walk.LineEdit
	var addressEdit *walk.LineEdit
	var cityEdit *walk.LineEdit
	var stateEdit *walk.LineEdit
	var zipEdit *walk.LineEdit
	var phoneEdit *walk.LineEdit
	var emailEdit *walk.LineEdit
	var err error
	err = declarative.Dialog{CancelButton: &cancelPB, MinSize: declarative.Size{Width: 400, Height: 300}, Layout: declarative.VBox{}, Children: []declarative.Widget{declarative.Composite{Layout: declarative.Grid{Columns: 2}, Children: []declarative.Widget{declarative.Label{Text: "First Name:"}, declarative.LineEdit{Text: (*c).FirstName, AssignTo: &firstNameEdit}, declarative.Label{Text: "Last Name:"}, declarative.LineEdit{AssignTo: &lastNameEdit, Text: (*c).LastName}, declarative.Label{Text: "Address:"}, declarative.LineEdit{AssignTo: &addressEdit, Text: (*c).Address}, declarative.Label{Text: "City:"}, declarative.LineEdit{AssignTo: &cityEdit, Text: (*c).City}, declarative.Label{Text: "State:"}, declarative.LineEdit{AssignTo: &stateEdit, Text: (*c).State}, declarative.Label{Text: "Zip:"}, declarative.LineEdit{AssignTo: &zipEdit, Text: (*c).Zip}, declarative.Label{Text: "Phone:"}, declarative.LineEdit{AssignTo: &phoneEdit, Text: (*c).Phone}, declarative.Label{Text: "Email:"}, declarative.LineEdit{AssignTo: &emailEdit, Text: (*c).Email}}}, declarative.Composite{Layout: declarative.HBox{}, Children: []declarative.Widget{declarative.HSpacer{}, declarative.PushButton{AssignTo: &acceptPB, Text: "OK"}, declarative.PushButton{Text: "Cancel", AssignTo: &cancelPB}}}}, AssignTo: &dlg, Title: title, DefaultButton: &acceptPB}.Create(gMainWindow)
	if (err != nil) {
		return false
	}
	(*acceptPB).Clicked().Attach(AcceptDialog)
	(*cancelPB).Clicked().Attach(CancelDialog)
	gDialogEdits.firstName = firstNameEdit
	gDialogEdits.lastName = lastNameEdit
	gDialogEdits.address = addressEdit
	gDialogEdits.city = cityEdit
	gDialogEdits.state = stateEdit
	gDialogEdits.zip = zipEdit
	gDialogEdits.phone = phoneEdit
	gDialogEdits.email = emailEdit
	gDialogEdits.contact = c
	gDialogEdits.dialog = dlg
	var result int = (*dlg).Run()
	return (result == walk.DlgCmdOK)
}

func AcceptDialog() {
	var c *Contact = gDialogEdits.contact
	(*c).FirstName = (*gDialogEdits.firstName).Text()
	(*c).LastName = (*gDialogEdits.lastName).Text()
	(*c).Address = (*gDialogEdits.address).Text()
	(*c).City = (*gDialogEdits.city).Text()
	(*c).State = (*gDialogEdits.state).Text()
	(*c).Zip = (*gDialogEdits.zip).Text()
	(*c).Phone = (*gDialogEdits.phone).Text()
	(*c).Email = (*gDialogEdits.email).Text()
	(*gDialogEdits.dialog).Accept()
}

func CancelDialog() {
	(*gDialogEdits.dialog).Cancel()
}

func Main() {
	var exePath string
	var err error
	exePath, err = os.Executable()
	var dbPath string = filepath.Join(filepath.Dir(exePath), "contacts.db")
	var db *sql.DB = InitDatabase(dbPath)
	if (db == nil) {
		walk.MsgBox(nil, "Error", "Failed to initialize database", walk.MsgBoxIconError)
		return
	}
	CreateContactsTable(db)
	SeedDatabase(db)
	gModel = new(ContactModel)
	(*gModel).DB = db
	(*gModel).SortCol = 1
	(*gModel).SortAsc = true
	(*gModel).Items = GetAllContacts(db, 1, true)
	var tableView *walk.TableView
	err = declarative.MainWindow{MenuItems: []declarative.MenuItem{declarative.Menu{Text: "&File", Items: []declarative.MenuItem{declarative.Action{Text: "&New Contact\tCtrl+N", OnTriggered: OnNewContact}, declarative.Action{Text: "&Edit Contact\tCtrl+E", OnTriggered: OnEditContact}, declarative.Action{Text: "&Delete Contact\tDel", OnTriggered: OnDeleteContact}, declarative.Separator{}, declarative.Action{OnTriggered: OnExit, Text: "E&xit\tAlt+F4"}}}}, Children: []declarative.Widget{declarative.TableView{Model: gModel, Columns: []declarative.TableViewColumn{declarative.TableViewColumn{Title: "First Name", Width: 100}, declarative.TableViewColumn{Title: "Last Name", Width: 100}, declarative.TableViewColumn{Title: "City", Width: 120}, declarative.TableViewColumn{Title: "State", Width: 50}, declarative.TableViewColumn{Title: "Phone", Width: 120}}, OnItemActivated: OnEditContact, AssignTo: &tableView}}, StatusBarItems: []declarative.StatusBarItem{declarative.StatusBarItem{AssignTo: &gStatusBar, Text: "Ready"}}, AssignTo: &gMainWindow, Title: "Contact Book", MinSize: declarative.Size{Width: 600, Height: 400}, Size: declarative.Size{Width: 800, Height: 600}, Layout: declarative.VBox{}}.Create()
	if (err != nil) {
		walk.MsgBox(nil, "Error", "Failed to create window", walk.MsgBoxIconError)
		return
	}
	(*gModel).TV = tableView
	UpdateStatus()
	(*gMainWindow).Run()
	(*db).Close()
}

func OnExit() {
	(*gMainWindow).Close()
}

func main() {
	Main()
}
