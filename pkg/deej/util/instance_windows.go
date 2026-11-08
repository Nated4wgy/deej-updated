package util
import("crypto/sha256";"fmt";"strings";"golang.org/x/sys/windows")
func AcquireInstance(directory string)(func(),error){sum:=sha256.Sum256([]byte(strings.ToLower(directory)));name,_:=windows.UTF16PtrFromString(fmt.Sprintf("Local\\Deej-%x",sum[:12]));handle,err:=windows.CreateMutex(nil,false,name);if err!=nil{if handle!=0{windows.CloseHandle(handle)};return nil,fmt.Errorf("deej is already running or cannot acquire its instance lock: %w",err)};return func(){windows.CloseHandle(handle)},nil}
