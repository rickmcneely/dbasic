' Form1 Code - Basic Widgets Demo
' Demonstrates Label, TextBox, and Button controls

SUB Form_Load()
    ' Initialize form
    lblTitle.Caption = "Basic Widgets Demo"
    lblMessage.Caption = ""
END SUB

SUB btnSubmit_Click()
    ' Handle Submit button click
    IF txtName.Value <> "" THEN
        lblMessage.Caption = "Hello, " & txtName.Value & "!"
    ELSE
        lblMessage.Caption = "Please enter your name"
    ENDIF
END SUB

SUB btnClear_Click()
    ' Handle Clear button click
    txtName.Value = ""
    txtEmail.Value = ""
    lblMessage.Caption = "Form cleared"
END SUB
