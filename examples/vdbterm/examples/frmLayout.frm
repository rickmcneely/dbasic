' frmLayout Code - Frame and ScrollBar Demo
' Demonstrates Frame (grouping), HScrollBar, and VScrollBar

SUB Form_Load()
    ' Initialize scrollbar values
    hsbVolume.Value = "50"
    hsbVolume.MinValue = 0
    hsbVolume.MaxValue = 100

    hsbBrightness.Value = "75"
    hsbBrightness.MinValue = 0
    hsbBrightness.MaxValue = 100

    vsbScroll.Value = "0"
    vsbScroll.MinValue = 0
    vsbScroll.MaxValue = 100

    ' Update display labels
    lblVolumeVal.Caption = "50"
    lblBrightVal.Caption = "75"
    lblScrollPos.Caption = "Position: 0"
END SUB

SUB hsbVolume_Change()
    ' Update volume display
    lblVolumeVal.Caption = hsbVolume.Value
END SUB

SUB hsbBrightness_Change()
    ' Update brightness display
    lblBrightVal.Caption = hsbBrightness.Value
END SUB

SUB vsbScroll_Change()
    ' Update scroll position display
    lblScrollPos.Caption = "Position: " & vsbScroll.Value
END SUB

SUB btnApply_Click()
    ' Apply settings
    lblContent.Caption = "Vol:" & hsbVolume.Value & " Br:" & hsbBrightness.Value
END SUB

SUB btnReset_Click()
    ' Reset to defaults
    hsbVolume.Value = "50"
    hsbBrightness.Value = "75"
    vsbScroll.Value = "0"
    lblVolumeVal.Caption = "50"
    lblBrightVal.Caption = "75"
    lblScrollPos.Caption = "Position: 0"
    lblContent.Caption = "Settings Reset"
END SUB
