' frmNew Code - New Widgets Demo
' Demonstrates TextArea, Progress, Spinner, Table, and Timer

SUB Form_Load()
    ' Initialize TextArea with sample content
    txtEditor.Value = "This is a multi-line text editor.\nYou can type multiple lines here.\nSupports scrolling and editing."
    txtEditor.ShowLineNumbers = TRUE

    ' Initialize Progress bar
    prgDownload.Value = "0"
    prgDownload.Total = 100
    prgDownload.ShowPercentage = TRUE
    lblPrgVal.Caption = "0%"

    ' Initialize Spinner
    spnLoading.SpinnerStyle = 0

    ' Initialize Table with columns and sample data
    tblData.Columns = "ID;Name;Status;Progress"
    tblData.Rows = "1;Task A;Running;45%|2;Task B;Complete;100%|3;Task C;Pending;0%"
    tblData.ShowHeaders = TRUE

    ' Timer for progress simulation (disabled initially)
    tmrProgress.Interval = 100
    tmrProgress.Enabled = FALSE
END SUB

SUB btnStart_Click()
    ' Start progress simulation
    prgDownload.Value = "0"
    tmrProgress.Enabled = TRUE
    spnLoading.Visible = TRUE
END SUB

SUB btnReset_Click()
    ' Reset everything
    prgDownload.Value = "0"
    lblPrgVal.Caption = "0%"
    tmrProgress.Enabled = FALSE
    spnLoading.Visible = FALSE
END SUB

SUB tmrProgress_Timer()
    ' Increment progress
    DIM current AS INTEGER = ParseInt(prgDownload.Value)
    IF current < 100 THEN
        current = current + 5
        prgDownload.Value = Str(current)
        lblPrgVal.Caption = Str(current) & "%"
    ELSE
        ' Complete
        tmrProgress.Enabled = FALSE
        spnLoading.Visible = FALSE
        lblPrgVal.Caption = "Complete!"
    ENDIF
END SUB

SUB txtEditor_Change()
    ' Handle text changes
    ' Could update character count, etc.
END SUB

SUB tblData_Click()
    ' Handle table row selection
    ' Could display row details
END SUB
