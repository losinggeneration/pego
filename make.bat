@echo off

:: Files of interest
setlocal
set SOURCE=peg.go match.go captures.go instructions.go
set OBJECT=peg.8
set PROGRAM=peg.exe

call :compileifmodified %PROGRAM% %SOURCE%
if errorlevel 1 exit /B 1
call :run %*
if errorlevel 1 exit /B 1
exit /B 0

:: compileifmodified <program> <source1> ... <sourceN>
:compileifmodified
if "%2"=="" goto :EOF
if "%~t1" LSS "%~t2" goto :loopend
shift /2
goto :compileifmodified
:loopend
echo %2 modified!
call :compile
if errorlevel 1 exit /B 1
call :link
if errorlevel 1 exit /B 1
exit /B 0

:: compile
:compile
echo Compiling...
8g -e -o %OBJECT% %SOURCE%
if errorlevel 1 goto :compileerror
if not errorlevel 0 goto :compileerror
exit /B 0
:compileerror
echo Compile failed. Exit code = %ERRORLEVEL%
exit /B 1

:: link
:link
echo Linking....
8l -o %PROGRAM% %OBJECT%
if errorlevel 1 goto :linkerror
if not errorlevel 0 goto :linkerror
if exist %OBJECT% del %OBJECT%
exit /B 0
:linkerror
echo Link failed. Exit code = %ERRORLEVEL%
exit /B 1

:: run <arg1> ... <argN>
:run
echo Running...
%PROGRAM% %*
if errorlevel 1 goto :runerror
if not errorlevel 0 goto :runerror
echo Program ended. Exit code = %ERRORLEVEL%
exit /B 0
:runerror
echo Program failed. Exit code = %ERRORLEVEL%
exit /B 1

