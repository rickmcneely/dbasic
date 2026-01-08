package main

import (
	"path/filepath"
	"github.com/lxn/walk"
	"github.com/lxn/walk/declarative"
	_ "modernc.org/sqlite"
	"database/sql"
	"fmt"
	"os"
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
	var seedContacts []Contact = []Contact{Contact{FirstName: "Herman", LastName: "Munster", Address: "1313 Mockingbird Lane", City: "Mockingbird Heights", State: "CA", Zip: "90210", Phone: "555-555-0001", Email: "herman.munster@mockingbird.example"}, Contact{City: "Westfield", State: "NJ", Zip: "07090", Phone: "555-555-0002", Email: "gomez.addams@cemetery.example", FirstName: "Gomez", LastName: "Addams", Address: "0001 Cemetery Lane"}, Contact{Address: "301 Cobblestone Way", City: "Bedrock", State: "", Zip: "00001", Phone: "555-555-0003", Email: "fred.flintstone@bedrock.example", FirstName: "Fred", LastName: "Flintstone"}, Contact{City: "Orbit City", State: "", Zip: "99999", Phone: "555-555-0004", Email: "george.jetson@spacely.example", FirstName: "George", LastName: "Jetson", Address: "Skypad Apartments"}, Contact{State: "", Zip: "49007", Phone: "555-555-0005", Email: "homer.simpson@springfield.example", FirstName: "Homer", LastName: "Simpson", Address: "742 Evergreen Terrace", City: "Springfield"}, Contact{FirstName: "Archie", LastName: "Bunker", Address: "704 Houser Street", City: "Queens", State: "NY", Zip: "11375", Phone: "555-555-0006", Email: "archie.bunker@queens.example"}, Contact{FirstName: "Al", LastName: "Bundy", Address: "9764 Jeopardy Lane", City: "Chicago", State: "IL", Zip: "60614", Phone: "555-555-0007", Email: "al.bundy@chicago.example"}, Contact{Zip: "12345", Phone: "555-555-0008", Email: "ward.cleaver@mayfield.example", FirstName: "Ward", LastName: "Cleaver", Address: "211 Pine Street", City: "Mayfield", State: ""}, Contact{Email: "mike.brady@losangeles.example", FirstName: "Mike", LastName: "Brady", Address: "4222 Clinton Way", City: "Los Angeles", State: "CA", Zip: "91604", Phone: "555-555-0009"}, Contact{City: "New York", State: "NY", Zip: "10065", Phone: "555-555-0010", Email: "ricky.ricardo@tropicana.example", FirstName: "Ricky", LastName: "Ricardo", Address: "623 East 68th Street"}, Contact{Zip: "11233", Phone: "555-555-0011", Email: "ralph.kramden@gotham.example", FirstName: "Ralph", LastName: "Kramden", Address: "328 Chauncey Street", City: "Brooklyn", State: "NY"}, Contact{State: "NY", Zip: "10028", Phone: "555-555-0012", Email: "george.jefferson@deluxe.example", FirstName: "George", LastName: "Jefferson", Address: "185 East 85th Street", City: "New York"}, Contact{Email: "cliff.huxtable@brooklyn.example", FirstName: "Cliff", LastName: "Huxtable", Address: "10 Stigwood Avenue", City: "Brooklyn", State: "NY", Zip: "11215", Phone: "555-555-0013"}, Contact{LastName: "Tanner", Address: "1882 Gerard Street", City: "San Francisco", State: "CA", Zip: "94115", Phone: "555-555-0014", Email: "danny.tanner@sanfrancisco.example", FirstName: "Danny"}, Contact{City: "Detroit", State: "MI", Zip: "48226", Phone: "555-555-0015", Email: "tim.taylor@tooltime.example", FirstName: "Tim", LastName: "Taylor", Address: "510 Glenview Road"}, Contact{FirstName: "Ray", LastName: "Barone", Address: "320 Fowler Street", City: "Lynbrook", State: "NY", Zip: "11563", Phone: "555-555-0016", Email: "ray.barone@newsday.example"}, Contact{Address: "3121 Aberdeen Street", City: "Queens", State: "NY", Zip: "11375", Phone: "555-555-0017", Email: "doug.heffernan@ips.example", FirstName: "Doug", LastName: "Heffernan"}, Contact{Phone: "555-555-0018", Email: "peter.griffin@quahog.example", FirstName: "Peter", LastName: "Griffin", Address: "31 Spooner Street", City: "Quahog", State: "RI", Zip: "02901"}, Contact{Address: "84 Rainey Street", City: "Arlen", State: "TX", Zip: "73104", Phone: "555-555-0019", Email: "hank.hill@strickland.example", FirstName: "Hank", LastName: "Hill"}, Contact{Address: "10336 Dunphy Lane", City: "Los Angeles", State: "CA", Zip: "90077", Phone: "555-555-0020", Email: "phil.dunphy@realestate.example", FirstName: "Phil", LastName: "Dunphy"}, Contact{Phone: "555-555-0021", Email: "bob.belcher@burgers.example", FirstName: "Bob", LastName: "Belcher", Address: "123 Ocean Avenue", City: "Seymours Bay", State: "NJ", Zip: "07001"}, Contact{Phone: "555-555-0022", Email: "walter.white@graymatter.example", FirstName: "Walter", LastName: "White", Address: "308 Negra Arroyo Lane", City: "Albuquerque", State: "NM", Zip: "87104"}, Contact{LastName: "Draper", Address: "783 Park Avenue", City: "New York", State: "NY", Zip: "10021", Phone: "555-555-0023", Email: "don.draper@scdp.example", FirstName: "Don"}, Contact{Email: "michael.scott@dundermifflin.example", FirstName: "Michael", LastName: "Scott", Address: "1725 Slough Avenue", City: "Scranton", State: "PA", Zip: "18503", Phone: "555-555-0024"}, Contact{FirstName: "Leslie", LastName: "Knope", Address: "314 Pawnee Way", City: "Pawnee", State: "IN", Zip: "47408", Phone: "555-555-0025", Email: "leslie.knope@pawnee.example"}}
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
	err = declarative.Dialog{CancelButton: &cancelPB, MinSize: declarative.Size{Width: 400, Height: 300}, Layout: declarative.VBox{}, Children: []declarative.Widget{declarative.Composite{Layout: declarative.Grid{Columns: 2}, Children: []declarative.Widget{declarative.Label{Text: "First Name:"}, declarative.LineEdit{Text: (*c).FirstName, AssignTo: &firstNameEdit}, declarative.Label{Text: "Last Name:"}, declarative.LineEdit{AssignTo: &lastNameEdit, Text: (*c).LastName}, declarative.Label{Text: "Address:"}, declarative.LineEdit{AssignTo: &addressEdit, Text: (*c).Address}, declarative.Label{Text: "City:"}, declarative.LineEdit{Text: (*c).City, AssignTo: &cityEdit}, declarative.Label{Text: "State:"}, declarative.LineEdit{AssignTo: &stateEdit, Text: (*c).State}, declarative.Label{Text: "Zip:"}, declarative.LineEdit{AssignTo: &zipEdit, Text: (*c).Zip}, declarative.Label{Text: "Phone:"}, declarative.LineEdit{AssignTo: &phoneEdit, Text: (*c).Phone}, declarative.Label{Text: "Email:"}, declarative.LineEdit{AssignTo: &emailEdit, Text: (*c).Email}}}, declarative.Composite{Layout: declarative.HBox{}, Children: []declarative.Widget{declarative.HSpacer{}, declarative.PushButton{AssignTo: &acceptPB, Text: "OK"}, declarative.PushButton{AssignTo: &cancelPB, Text: "Cancel"}}}}, AssignTo: &dlg, Title: title, DefaultButton: &acceptPB}.Create(gMainWindow)
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
	err = declarative.MainWindow{Title: "Contact Book", MinSize: declarative.Size{Width: 600, Height: 400}, Size: declarative.Size{Width: 800, Height: 600}, Layout: declarative.VBox{}, MenuItems: []declarative.MenuItem{declarative.Menu{Text: "&File", Items: []declarative.MenuItem{declarative.Action{Text: "&New Contact\tCtrl+N", OnTriggered: OnNewContact}, declarative.Action{Text: "&Edit Contact\tCtrl+E", OnTriggered: OnEditContact}, declarative.Action{OnTriggered: OnDeleteContact, Text: "&Delete Contact\tDel"}, declarative.Separator{}, declarative.Action{Text: "E&xit\tAlt+F4", OnTriggered: OnExit}}}}, Children: []declarative.Widget{declarative.TableView{AssignTo: &tableView, Model: gModel, Columns: []declarative.TableViewColumn{declarative.TableViewColumn{Title: "First Name", Width: 100}, declarative.TableViewColumn{Title: "Last Name", Width: 100}, declarative.TableViewColumn{Title: "City", Width: 120}, declarative.TableViewColumn{Width: 50, Title: "State"}, declarative.TableViewColumn{Title: "Phone", Width: 120}}, OnItemActivated: OnEditContact}}, StatusBarItems: []declarative.StatusBarItem{declarative.StatusBarItem{AssignTo: &gStatusBar, Text: "Ready"}}, AssignTo: &gMainWindow}.Create()
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
