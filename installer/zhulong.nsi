; Zhulong Windows 安装包配置 (NSIS)
!define PRODUCT_NAME "Zhulong"
!define PRODUCT_VERSION "1.0.0"
!define PRODUCT_PUBLISHER "qoqu"

Name "${PRODUCT_NAME} ${PRODUCT_VERSION}"
OutFile "zhulong-setup-${PRODUCT_VERSION}.exe"
InstallDir "$PROGRAMFILES\${PRODUCT_NAME}"

Section "Install"
  SetOutPath "$INSTDIR"
  
  ; 主程序
  File "..\build\zhulong.exe"
  File "..\build\zhulong-desktop.exe"
  
  ; 配置文件
  File /r "..\config\*.*"
  
  ; 创建快捷方式
  CreateShortCut "$SMPROGRAMS\Zhulong.lnk" "$INSTDIR\zhulong-desktop.exe"
  CreateShortCut "$DESKTOP\Zhulong.lnk" "$INSTDIR\zhulong-desktop.exe"
  
  ; 写注册表
  WriteUninstaller "$INSTDIR\uninstall.exe"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" "DisplayName" "${PRODUCT_NAME}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" "UninstallString" "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\*.*"
  RMDir "$INSTDIR"
  Delete "$SMPROGRAMS\Zhulong.lnk"
  Delete "$DESKTOP\Zhulong.lnk"
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"
SectionEnd
