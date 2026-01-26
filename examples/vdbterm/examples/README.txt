VDBTerm Widget Examples
=======================

This directory contains VDBTerm project files demonstrating all widget types.
Open these projects in VDBTerm to see the widget layouts and code.

To open a project:
  ./vdbterm
  File -> Open Project -> select .vdbp file

Example Projects:
-----------------

1. basic_widgets.vdbp (Form1.frm)
   - Label (type 2): Static text display
   - TextBox (type 3): Single-line text input
   - Button (type 1): Clickable action buttons
   Demonstrates a simple form with name/email input and submit/clear buttons.

2. checkbox_option.vdbp (frmOptions.frm)
   - CheckBox (type 4): Toggle options (multiple can be selected)
   - Option (type 5): Radio buttons (single selection per group)
   - Frame (type 8): Visual grouping container
   Demonstrates text formatting options with style, color, and size groups.

3. list_controls.vdbp (frmLists.frm)
   - ListBox (type 6): Scrollable list with multiple items
   - ComboBox (type 7): Dropdown selection
   Demonstrates list management with add/remove functionality.

4. layout_controls.vdbp (frmLayout.frm)
   - Frame (type 8): Container with border and caption
   - HScrollBar (type 9): Horizontal scroll/slider control
   - VScrollBar (type 10): Vertical scroll/slider control
   Demonstrates settings panel with volume/brightness sliders.

5. new_widgets.vdbp (frmNew.frm)
   - TextArea (type 12): Multi-line text editor
   - Progress (type 13): Progress bar indicator
   - Spinner (type 14): Loading/activity spinner
   - Table (type 15): Data grid with columns and rows
   - Timer (type 11): Non-visible timer for animations
   Demonstrates the new BubbleTea-based widgets.

6. timer_demo.vdbp (frmTimer.frm)
   - Timer (type 11): Non-visible periodic event trigger
   - Progress (type 13): Animated progress bar
   - Spinner (type 14): Activity indicator
   Demonstrates clock, counter, and animation using timers.

7. all_widgets.vdbp (frmMain.frm)
   Comprehensive showcase of ALL widget types in a single form.
   Great for testing and as a reference for widget capabilities.

Widget Type Reference:
----------------------
Type 0:  Form (container)
Type 1:  Button
Type 2:  Label
Type 3:  TextBox (single-line input)
Type 4:  CheckBox (toggle)
Type 5:  Option (radio button)
Type 6:  ListBox (scrollable list)
Type 7:  ComboBox (dropdown)
Type 8:  Frame (grouping container)
Type 9:  HScrollBar (horizontal slider)
Type 10: VScrollBar (vertical slider)
Type 11: Timer (non-visible, triggers events)
Type 12: TextArea (multi-line editor)
Type 13: Progress (progress bar)
Type 14: Spinner (loading indicator)
Type 15: Table (data grid)
