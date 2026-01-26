' frmLists Code - ListBox and ComboBox Demo
' Demonstrates ListBox (scrollable list) and ComboBox (dropdown)

SUB Form_Load()
    ' Initialize ListBox with fruit items (semicolon-separated)
    lstFruits.Items = "Apple;Banana;Cherry;Date;Elderberry;Fig;Grape"

    ' Initialize ComboBox with country items
    cboCountry.Items = "USA;Canada;Mexico;UK;France;Germany;Japan"

    lblSelected.Caption = "Selected: (none)"
END SUB

SUB lstFruits_Click()
    ' Handle ListBox selection
    lblSelected.Caption = "Selected Fruit: " & lstFruits.Value
END SUB

SUB cboCountry_Change()
    ' Handle ComboBox selection
    lblSelected.Caption = "Selected Country: " & cboCountry.Value
END SUB

SUB btnAdd_Click()
    ' Add new item to ListBox
    IF txtNewItem.Value <> "" THEN
        lstFruits.Items = lstFruits.Items & ";" & txtNewItem.Value
        txtNewItem.Value = ""
        lblSelected.Caption = "Added new item to list"
    ENDIF
END SUB

SUB btnRemove_Click()
    ' Remove selected item from ListBox
    IF lstFruits.Value <> "" THEN
        lblSelected.Caption = "Removed: " & lstFruits.Value
        ' Note: Actual removal would require parsing Items string
    ENDIF
END SUB
