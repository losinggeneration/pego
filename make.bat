@echo off

:: Files of interest
setlocal
set SOURCE=peg.go captures.go instructions.go
set OBJECT=peg.8
set PROGRAM=peg.exe

set ABORT=

call :compileifmodified
if not "%ABORT%"=="" GOTO :EOF
call :run %*
goto :EOF

:: compileifmodified
:compileifmodified
call :compile %1 %2
if not "%ABORT%"=="" GOTO :EOF
call :link %2 %3
goto :EOF

:: compile <source> <object>
:compile
echo Compiling...
8g -o %OBJECT% %SOURCE%
if errorlevel 1 goto compileerror
if not errorlevel 0 goto compileerror
goto :EOF
:compileerror
echo Compile failed. Exit code = %ERRORLEVEL%
set ABORT=1
goto :EOF

:: link
:link
echo Linking....
8l -o %PROGRAM% %OBJECT%
if errorlevel 1 goto :linkerror
if not errorlevel 0 goto :linkerror
if exist %OBJECT% del %OBJECT%
goto :EOF
:linkerror
echo Link failed. Exit code = %ERRORLEVEL%
set ABORT=1
goto :EOF

:: run <arg1> ... <argN>
:run
echo Running...
%PROGRAM% %*
if errorlevel 1 goto :runerror
if not errorlevel 0 goto :runerror
echo Program ended. Exit code = %ERRORLEVEL%
goto :EOF
:runerror
echo Program failed. Exit code = %ERRORLEVEL%
set ABORT=1
goto :EOF

