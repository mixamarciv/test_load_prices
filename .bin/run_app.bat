﻿::получаем curpath:
@FOR /f %%i IN ("%0") DO SET curpath=%~dp0
::задаем основные переменные окружения
@CALL "%curpath%/set_path.bat"


@del app.exe
@CLS

@echo === build =====================================================================
go build -o app.exe

@echo ==== start ====================================================================
::app.exe
:: >> app.exe.log 2>&1

SET start_from=0
SET load_count=200
SET load_to=2147483646

for /l %%i in (%start_from%,%load_count%,%load_to%) do (
	app.exe --load_from %%i --load_count %load_count%
)

@echo ==== end ======================================================================
@PAUSE
