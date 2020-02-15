#NoEnv  ; Recommended for performance and compatibility with future AutoHotkey releases.
#Warn  ; Enable warnings to assist with detecting common errors.
SendMode Input  ; Recommended for new scripts due to its superior speed and reliability.
SetWorkingDir %A_ScriptDir%  ; Ensures a consistent starting directory.

#SingleInstance Ignore
;#NoTrayIcon
Gui, New, -SysMenu +Resize ;+MinSize640x480 +DPIScale
Gui, Add, Button, x250, Exit
Gui, Add, Text, xm, port: (default 10001)
Gui, Add, Edit, vPort, 10001
Gui, Add, Text
Gui, Add, Button, gButtonMimaGo, mima-go
Gui, Add, Button, gButtonShowMimaGo, show window
Gui, Add, Button, gButtonHideMimaGo, hide window
Gui, Show, W300 H200
Return

ButtonMimaGo:
Gui, Submit, NoHide
if Port not between 80 and 65536
    Port = 10001
Run, cmd.exe,,, mimaGoPID
WinWaitActive, ahk_pid %mimaGoPID%
Send mima-go.exe -port %Port%{enter}
; Run, mima-go.exe -port %Port%,, , mimaGoPID
Return

ButtonShowMimaGo:
WinShow, ahk_pid %mimaGoPID%
Return

ButtonHideMimaGo:
WinHide, ahk_pid %mimaGoPID%
Return

ButtonExit:
GuiClose:
DetectHiddenWindows, on
WinKill, ahk_pid %mimaGoPID%
ExitApp
