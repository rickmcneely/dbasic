' frmMain Code - All Widgets Showcase
' Comprehensive demo of all VDBTerm widget types

SUB Form_Load()
    ' Initialize Label (type 2) - already set via Caption property
    lblTitle.Caption = "VDBTerm Widget Showcase"

    ' Initialize TextBox (type 3)
    txtName.Value = ""
    txtName.Placeholder = "Enter your name"

    ' Initialize CheckBox (type 4)
    chkActive.Value = "1"
    chkNotify.Value = "0"

    ' Initialize Option/Radio (type 5)
    optLow.Value = "0"
    optMed.Value = "1"
    optHigh.Value = "0"

    ' Initialize ComboBox (type 7)
    cboCategory.Items = "General;Technical;Support;Sales;Other"

    ' Initialize ListBox (type 6)
    lstItems.Items = "Item 1;Item 2;Item 3;Item 4;Item 5"
    lstItems.ShowTitle = TRUE
    lstItems.Title = "Select Item"

    ' Initialize TextArea (type 12)
    txtEditor.Value = "Multi-line text editor.\nEdit content here."
    txtEditor.ShowLineNumbers = TRUE

    ' Initialize Progress (type 13)
    prgStatus.Value = "0"
    prgStatus.Total = 100
    prgStatus.ShowPercentage = TRUE

    ' Initialize Spinner (type 14)
    spnWorking.SpinnerStyle = 1
    spnWorking.Visible = FALSE

    ' Initialize Table (type 15)
    tblResults.Columns = "ID;Status;Value"
    tblResults.Rows = ""
    tblResults.ShowHeaders = TRUE

    ' Initialize Timer (type 11) - non-visible
    tmrUpdate.Interval = 500
    tmrUpdate.Enabled = FALSE
END SUB

SUB btnOK_Click()
    ' Demonstrate gathering form data
    DIM result AS STRING = "Name: " & txtName.Value
    result = result & "\nActive: " & chkActive.Value
    result = result & "\nNotify: " & chkNotify.Value

    IF optLow.Value = "1" THEN
        result = result & "\nPriority: Low"
    ELSEIF optMed.Value = "1" THEN
        result = result & "\nPriority: Medium"
    ELSE
        result = result & "\nPriority: High"
    ENDIF

    txtEditor.Value = result

    ' Start progress demo
    prgStatus.Value = "0"
    spnWorking.Visible = TRUE
    tmrUpdate.Enabled = TRUE
END SUB

SUB btnCancel_Click()
    ' Reset form
    txtName.Value = ""
    chkActive.Value = "0"
    chkNotify.Value = "0"
    optMed.Value = "1"
    optLow.Value = "0"
    optHigh.Value = "0"
    prgStatus.Value = "0"
    spnWorking.Visible = FALSE
    tmrUpdate.Enabled = FALSE
    txtEditor.Value = ""
    tblResults.Rows = ""
END SUB

SUB optLow_Click()
    optMed.Value = "0"
    optHigh.Value = "0"
END SUB

SUB optMed_Click()
    optLow.Value = "0"
    optHigh.Value = "0"
END SUB

SUB optHigh_Click()
    optLow.Value = "0"
    optMed.Value = "0"
END SUB

SUB lstItems_Click()
    ' Add selected item to table
    DIM rowCount AS INTEGER = CountRows(tblResults.Rows)
    DIM newRow AS STRING = Str(rowCount + 1) & ";Selected;" & lstItems.Value
    IF tblResults.Rows = "" THEN
        tblResults.Rows = newRow
    ELSE
        tblResults.Rows = tblResults.Rows & "|" & newRow
    ENDIF
END SUB

SUB tmrUpdate_Timer()
    ' Animate progress bar
    DIM current AS INTEGER = ParseInt(prgStatus.Value)
    IF current < 100 THEN
        current = current + 10
        prgStatus.Value = Str(current)
    ELSE
        tmrUpdate.Enabled = FALSE
        spnWorking.Visible = FALSE
        lblProgress.Caption = "Complete!"
    ENDIF
END SUB

FUNCTION CountRows(rows AS STRING) AS INTEGER
    IF rows = "" THEN
        RETURN 0
    ENDIF
    DIM count AS INTEGER = 1
    DIM i AS INTEGER
    FOR i = 1 TO Len(rows)
        IF Mid(rows, i, 1) = "|" THEN
            count = count + 1
        ENDIF
    NEXT
    RETURN count
END FUNCTION
