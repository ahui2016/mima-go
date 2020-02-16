#NoEnv  ; Recommended for performance and compatibility with future AutoHotkey releases.
#Warn  ; Enable warnings to assist with detecting common errors.
SendMode Input  ; Recommended for new scripts due to its superior speed and reliability.
SetWorkingDir %A_ScriptDir%  ; Ensures a consistent starting directory.

#SingleInstance Ignore
#NoTrayIcon
Run, cmd.exe,, Hide, mimaGoPID
Gui, New, +DPIScale +Resize +MinSize640x480 ;-SysMenu 
Gui, Font, s16
Gui, Add, Text,, mima-go 启动器
Gui, Add, Text
Gui, Add, Text, xm section, port: 
Gui, Add, Edit, ys vPort, 10001
Gui, Add, Text, ys, (default 10001)
Gui, Add, Text, xm, 点击 Start 即可启动程序, 控制台窗口会自动隐藏.
Gui, Add, Button, xm section vStartButton, Start
Gui, Add, Button, ys vRestartButton, Restart
GuiControl, Hide, RestartButton
Gui, Add, Link, xm vLinkToLocal
Gui, Add, Text, xm
Gui, Add, Text,, 点击 Show Console 可查看控制台信息 (比如出错信息).
Gui, Add, Button, xm section, Show Console
Gui, Add, Button, ys, Hide Console
Gui, Add, Text, xm section
Gui, Add, Button, x550, Exit
Gui, Show, W640 H480
Return

ButtonStart:
Gui, Submit, NoHide
if Port not between 80 and 65536
    Port = 10001
WinShow, ahk_pid %mimaGoPID%
WinWaitActive, ahk_pid %mimaGoPID%
; Run, mima-go.exe -port %Port%,, , mimaGoPID
Send mima-go.exe -port %Port%{enter}
Sleep, 500
WinHide, ahk_pid %mimaGoPID%
GuiControl, Show, RestartButton
GuiControl, Disable, StartButton
GuiControl, Text, LinkToLocal, 点击链接进入程序界面: <a href="http://localhost:%Port%">http://localhost:%Port%</a>
GuiControl, MoveDraw, LinkToLocal, w640
Return

ButtonRestart:
Gui, Submit, NoHide
if Port not between 80 and 65536
    Port = 10001
DetectHiddenWindows, on
WinKill, ahk_pid %mimaGoPID%
WinWaitClose, ahk_pid %mimaGoPID%
Run, cmd.exe,,, mimaGoPID
WinWaitActive, ahk_pid %mimaGoPID%
Send mima-go.exe -port %Port%{enter}
Sleep, 500
WinHide, ahk_pid %mimaGoPID%
DetectHiddenWindows, off
GuiControl, Text, LinkToLocal, 点击链接进入程序界面: <a href="http://localhost:%Port%">http://localhost:%Port%</a>
Return

ButtonShowConsole:
WinShow, ahk_pid %mimaGoPID%
Return

ButtonHideConsole:
WinHide, ahk_pid %mimaGoPID%
Return

ButtonExit:
GuiClose:
DetectHiddenWindows, on
WinKill, ahk_pid %mimaGoPID%
ExitApp
