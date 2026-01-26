' frmOptions Code - CheckBox and Option Demo
' Demonstrates CheckBox (multiple selection) and Option (single selection)

SUB Form_Load()
    ' Initialize - select defaults
    optRed.Value = "1"
    optMedium.Value = "1"
    UpdatePreview()
END SUB

SUB chkBold_Click()
    UpdatePreview()
END SUB

SUB chkItalic_Click()
    UpdatePreview()
END SUB

SUB chkUnderline_Click()
    UpdatePreview()
END SUB

SUB optRed_Click()
    optGreen.Value = "0"
    optBlue.Value = "0"
    UpdatePreview()
END SUB

SUB optGreen_Click()
    optRed.Value = "0"
    optBlue.Value = "0"
    UpdatePreview()
END SUB

SUB optBlue_Click()
    optRed.Value = "0"
    optGreen.Value = "0"
    UpdatePreview()
END SUB

SUB optSmall_Click()
    optMedium.Value = "0"
    optLarge.Value = "0"
    UpdatePreview()
END SUB

SUB optMedium_Click()
    optSmall.Value = "0"
    optLarge.Value = "0"
    UpdatePreview()
END SUB

SUB optLarge_Click()
    optSmall.Value = "0"
    optMedium.Value = "0"
    UpdatePreview()
END SUB

SUB UpdatePreview()
    DIM preview AS STRING = "Preview: "
    DIM style AS STRING = ""

    IF chkBold.Value = "1" THEN
        style = style & "[B]"
    ENDIF
    IF chkItalic.Value = "1" THEN
        style = style & "[I]"
    ENDIF
    IF chkUnderline.Value = "1" THEN
        style = style & "[U]"
    ENDIF

    DIM color AS STRING = ""
    IF optRed.Value = "1" THEN
        color = "RED"
    ELSEIF optGreen.Value = "1" THEN
        color = "GREEN"
    ELSE
        color = "BLUE"
    ENDIF

    DIM size AS STRING = ""
    IF optSmall.Value = "1" THEN
        size = "sm"
    ELSEIF optMedium.Value = "1" THEN
        size = "Medium"
    ELSE
        size = "LARGE"
    ENDIF

    lblPreview.Caption = preview & style & " " & color & " " & size
END SUB
