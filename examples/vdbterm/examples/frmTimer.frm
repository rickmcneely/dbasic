' frmTimer Code - Timer Control Demo
' Demonstrates Timer for animations and periodic updates

DIM seconds AS INTEGER
DIM counter AS INTEGER
DIM animProgress AS INTEGER
DIM animDirection AS INTEGER

SUB Form_Load()
    ' Initialize counters
    seconds = 0
    counter = 0
    animProgress = 0
    animDirection = 1

    ' Initialize display
    lblTime.Caption = "00:00:00"
    lblCounter.Caption = "0"

    ' Initialize progress bar
    prgAnim.Value = "0"
    prgAnim.Total = 100

    ' Initialize spinner
    spnAnim.SpinnerStyle = 2
    spnAnim.Visible = FALSE

    ' Clock timer - fires every 1 second
    tmrClock.Interval = 1000
    tmrClock.Enabled = FALSE

    ' Animation timer - fires every 50ms for smooth animation
    tmrAnim.Interval = 50
    tmrAnim.Enabled = FALSE
END SUB

SUB btnStart_Click()
    ' Start both timers
    tmrClock.Enabled = TRUE
    tmrAnim.Enabled = TRUE
    spnAnim.Visible = TRUE
END SUB

SUB btnStop_Click()
    ' Stop both timers
    tmrClock.Enabled = FALSE
    tmrAnim.Enabled = FALSE
    spnAnim.Visible = FALSE
END SUB

SUB btnReset_Click()
    ' Reset everything
    seconds = 0
    counter = 0
    animProgress = 0
    animDirection = 1

    lblTime.Caption = "00:00:00"
    lblCounter.Caption = "0"
    prgAnim.Value = "0"

    tmrClock.Enabled = FALSE
    tmrAnim.Enabled = FALSE
    spnAnim.Visible = FALSE
END SUB

SUB tmrClock_Timer()
    ' Increment elapsed time
    seconds = seconds + 1
    counter = counter + 1

    ' Format time as HH:MM:SS
    DIM hrs AS INTEGER = seconds / 3600
    DIM mins AS INTEGER = (seconds MOD 3600) / 60
    DIM secs AS INTEGER = seconds MOD 60

    DIM timeStr AS STRING = ""
    IF hrs < 10 THEN
        timeStr = "0"
    ENDIF
    timeStr = timeStr & Str(hrs) & ":"

    IF mins < 10 THEN
        timeStr = timeStr & "0"
    ENDIF
    timeStr = timeStr & Str(mins) & ":"

    IF secs < 10 THEN
        timeStr = timeStr & "0"
    ENDIF
    timeStr = timeStr & Str(secs)

    lblTime.Caption = timeStr
    lblCounter.Caption = Str(counter)
END SUB

SUB tmrAnim_Timer()
    ' Animate progress bar back and forth
    animProgress = animProgress + (animDirection * 2)

    IF animProgress >= 100 THEN
        animProgress = 100
        animDirection = -1
    ELSEIF animProgress <= 0 THEN
        animProgress = 0
        animDirection = 1
    ENDIF

    prgAnim.Value = Str(animProgress)
END SUB
