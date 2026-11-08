package deej
import("errors";ole "github.com/go-ole/go-ole")
func initializeAudioThread()error{err:=ole.CoInitializeEx(0,ole.COINIT_MULTITHREADED);var e *ole.OleError;if errors.As(err,&e)&&e.Code()==1{return nil};return err}
func uninitializeAudioThread(){ole.CoUninitialize()}
