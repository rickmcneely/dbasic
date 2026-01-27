' frmMain Code - All Widgets Showcase
' Comprehensive demo of all VDBTerm widget types

DIM prgDirection AS INTEGER

SUB Form_Load()
    ' Initialize title
    lblTitle.Caption = "VDBTerm Widget Showcase"

    ' Add items to listbox
    lstItems.AddItem("Apple")
    lstItems.AddItem("Banana")
    lstItems.AddItem("Cherry")
    lstItems.AddItem("Date")
    lstItems.AddItem("Elderberry")

    ' Set default option
    optLow.Selected = TRUE

    ' Set initial progress and direction
    prgStatus.Value = 0
    prgDirection = 5
END SUB

SUB tmrUpdate_Timer()
    ' Astable mode - ride the progress bar up and down
    prgStatus.Value = prgStatus.Value + prgDirection

    ' Reverse direction at boundaries
    IF prgStatus.Value >= 100 THEN
        prgStatus.Value = 100
        prgDirection = -5
    ELSEIF prgStatus.Value <= 0 THEN
        prgStatus.Value = 0
        prgDirection = 5
    ENDIF

    ' Debug: show current value in title
    lblTitle.Caption = "TICK! " & Str(prgStatus.Value) & "%"
END SUB

SUB btnOK_Click()
    ' Show current form state
    DIM msg AS STRING = "Name: " & txtName.Text
    IF chkActive.Checked THEN
        msg = msg & " [Active]"
    ENDIF
    IF chkNotify.Checked THEN
        msg = msg & " [Notify]"
    ENDIF
    IF optLow.Selected THEN
        msg = msg & " (Low)"
    ELSEIF optMed.Selected THEN
        msg = msg & " (Med)"
    ELSE
        msg = msg & " (High)"
    ENDIF
    IF lstItems.SelectedIndex >= 0 THEN
        msg = msg & " Item: " & lstItems.SelectedItem
    ENDIF
    lblTitle.Caption = msg
END SUB

SUB btnCancel_Click()
    ' Reset the form
    txtName.Text = ""
    chkActive.Checked = FALSE
    chkNotify.Checked = FALSE
    optLow.Selected = TRUE
    optMed.Selected = FALSE
    optHigh.Selected = FALSE
    lstItems.SelectedIndex = 0
    prgStatus.Value = 0
    prgDirection = 5
    lblTitle.Caption = "Form Reset!"
END SUB

SUB chkActive_Click()
    IF chkActive.Checked THEN
        lblTitle.Caption = "TIMER STARTING..."
        tmrUpdate.Enabled = TRUE
    ELSE
        lblTitle.Caption = "TIMER STOPPED"
        tmrUpdate.Enabled = FALSE
    ENDIF
END SUB

SUB chkNotify_Click()
    IF chkNotify.Checked THEN
        lblTitle.Caption = "Notifications enabled"
    ELSE
        lblTitle.Caption = "Notifications disabled"
    ENDIF
END SUB

SUB optLow_Click()
    lblTitle.Caption = "Priority: Low"
    prgStatus.Value = 25
END SUB

SUB optMed_Click()
    lblTitle.Caption = "Priority: Medium"
    prgStatus.Value = 50
END SUB

SUB optHigh_Click()
    lblTitle.Caption = "Priority: High"
    prgStatus.Value = 100
END SUB

SUB lstItems_Click()
    IF lstItems.SelectedIndex >= 0 THEN
        lblTitle.Caption = "Selected: " & lstItems.SelectedItem
    ENDIF
END SUB
