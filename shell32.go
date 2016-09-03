// Copyright 2010-2012 The W32 Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package w32

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

var (
	modshell32 = syscall.NewLazyDLL("shell32.dll")

	procSHBrowseForFolder   = modshell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDList = modshell32.NewProc("SHGetPathFromIDListW")
	procDragAcceptFiles     = modshell32.NewProc("DragAcceptFiles")
	procDragQueryFile       = modshell32.NewProc("DragQueryFileW")
	procDragQueryPoint      = modshell32.NewProc("DragQueryPoint")
	procDragFinish          = modshell32.NewProc("DragFinish")
	procShellExecute        = modshell32.NewProc("ShellExecuteW")
	procExtractIcon         = modshell32.NewProc("ExtractIconW")
	// add
	procShellNotifyIcon = modshell32.NewProc("Shell_NotifyIconW")
)

func SHBrowseForFolder(bi *BROWSEINFO) uintptr {
	ret, _, _ := procSHBrowseForFolder.Call(uintptr(unsafe.Pointer(bi)))

	return ret
}

func SHGetPathFromIDList(idl uintptr) string {
	buf := make([]uint16, 1024)
	procSHGetPathFromIDList.Call(
		idl,
		uintptr(unsafe.Pointer(&buf[0])))

	return syscall.UTF16ToString(buf)
}

func DragAcceptFiles(hwnd HWND, accept bool) {
	procDragAcceptFiles.Call(
		uintptr(hwnd),
		uintptr(BoolToBOOL(accept)))
}

func DragQueryFile(hDrop HDROP, iFile uint) (fileName string, fileCount uint) {
	ret, _, _ := procDragQueryFile.Call(
		uintptr(hDrop),
		uintptr(iFile),
		0,
		0)

	fileCount = uint(ret)

	if iFile != 0xFFFFFFFF {
		buf := make([]uint16, fileCount+1)

		ret, _, _ := procDragQueryFile.Call(
			uintptr(hDrop),
			uintptr(iFile),
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(fileCount+1))

		if ret == 0 {
			panic("Invoke DragQueryFile error.")
		}

		fileName = syscall.UTF16ToString(buf)
	}

	return
}

func DragQueryPoint(hDrop HDROP) (x, y int, isClientArea bool) {
	var pt POINT
	ret, _, _ := procDragQueryPoint.Call(
		uintptr(hDrop),
		uintptr(unsafe.Pointer(&pt)))

	return int(pt.X), int(pt.Y), (ret == 1)
}

func DragFinish(hDrop HDROP) {
	procDragFinish.Call(uintptr(hDrop))
}

func ShellExecute(hwnd HWND, lpOperation, lpFile, lpParameters, lpDirectory string, nShowCmd int) error {
	var op, param, directory uintptr
	if len(lpOperation) != 0 {
		op = uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(lpOperation)))
	}
	if len(lpParameters) != 0 {
		param = uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(lpParameters)))
	}
	if len(lpDirectory) != 0 {
		directory = uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(lpDirectory)))
	}

	ret, _, _ := procShellExecute.Call(
		uintptr(hwnd),
		op,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(lpFile))),
		param,
		directory,
		uintptr(nShowCmd))

	errorMsg := ""
	if ret != 0 && ret <= 32 {
		switch int(ret) {
		case ERROR_FILE_NOT_FOUND:
			errorMsg = "The specified file was not found."
		case ERROR_PATH_NOT_FOUND:
			errorMsg = "The specified path was not found."
		case ERROR_BAD_FORMAT:
			errorMsg = "The .exe file is invalid (non-Win32 .exe or error in .exe image)."
		case SE_ERR_ACCESSDENIED:
			errorMsg = "The operating system denied access to the specified file."
		case SE_ERR_ASSOCINCOMPLETE:
			errorMsg = "The file name association is incomplete or invalid."
		case SE_ERR_DDEBUSY:
			errorMsg = "The DDE transaction could not be completed because other DDE transactions were being processed."
		case SE_ERR_DDEFAIL:
			errorMsg = "The DDE transaction failed."
		case SE_ERR_DDETIMEOUT:
			errorMsg = "The DDE transaction could not be completed because the request timed out."
		case SE_ERR_DLLNOTFOUND:
			errorMsg = "The specified DLL was not found."
		case SE_ERR_NOASSOC:
			errorMsg = "There is no application associated with the given file name extension. This error will also be returned if you attempt to print a file that is not printable."
		case SE_ERR_OOM:
			errorMsg = "There was not enough memory to complete the operation."
		case SE_ERR_SHARE:
			errorMsg = "A sharing violation occurred."
		default:
			errorMsg = fmt.Sprintf("Unknown error occurred with error code %v", ret)
		}
	} else {
		return nil
	}

	return errors.New(errorMsg)
}

func ExtractIcon(lpszExeFileName string, nIconIndex int) HICON {
	ret, _, _ := procExtractIcon.Call(
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(lpszExeFileName))),
		uintptr(nIconIndex))

	return HICON(ret)
}

// 注意：对于Windows 2000(Shell32.dll version 5.0)，Shell_NotifyIcon对于鼠标和键盘事件的处理与早期的操作系统的不同点在于：
// （1）用户使用键盘选择了通知图标的快捷菜单，Shell将发送WM_CONTEXTMENU消息给图标对应的应用程序，而早期操作系统则发送WM_RBUTTONDOWN和WM_RBUTTONUP消息；
// （2）用户使用键盘选择通知图标，并使用空格键或Enter键激活它，则Shell将发送NIN_KEYSELECT通知给应用程序，而早期版本则发送WM_RBUTTONDOWN和WM_RBUTTONUP消息；
// （3）用户使用鼠标选择通知图标，并使用Enter键激活它，Shell将发送NIN_SELECT通知给应用程序，而早期版本发送WM_RBUTTONDOWN和WM_RBUTTONUP消息；
// 对于Windows XP(Shell32.dll version 6.0)，当用户将鼠标指向关联着气泡通知的图标时，Shell将发送下列消息：
// （1）NIN_BALLOONSHOW：当气泡显示时发送（气泡在队列中排队）；
// （2）NIN_BALLOONHIDE：当气泡消失时发送，例如，当图标删除时。在气泡因为超时或者用户鼠标单击后消失时，不发送该消息；
// （3）NIN_BALLOONTIMEOUT：气泡超时后消失时发送；
// （4）NIN_BALLOONUSERCLICK：用户鼠标单击气泡使气泡消失时发送；
func Shell_NotifyIcon(message uint32, nid *NOTIFYICONDATA) bool {
	ret, _, _ := procShellNotifyIcon.Call(
		uintptr(message),
		uintptr(unsafe.Pointer(nid)),
	)
	return ret != 0
}

