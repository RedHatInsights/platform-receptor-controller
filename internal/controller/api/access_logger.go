package api

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"

	"github.com/gorilla/handlers"
	"github.com/sirupsen/logrus"
)

/*
type LogFormatter func(writer io.Writer, params LogFormatterParams)

type LogFormatterParams struct {
    Request    *http.Request
    URL        url.URL
    TimeStamp  time.Time
    StatusCode int
    Size       int
}


func CustomLoggingHandler(out io.Writer, h http.Handler, f LogFormatter) http.Handler


params.Request:&{Method:POST URL:/job Proto:HTTP/1.1 ProtoMajor:1 ProtoMinor:1 Header:map[Accept:[] Content-Length:[
155] Content-Type:[application/x-www-form-urlencoded] User-Agent:[curl/7.66.0] X-Rh-Identity:[eyJpZGVudGl0eSI6IHsiYWNjb
3VudF9udW1iZXIiOiAiMDAwMDAwMSIsICJpbnRlcm5hbCI6IHsib3JnX2lkIjogIjAwMDAwMSJ9fX0=]] Body:0xc0000ca1c0 GetBody:<nil> Conte
ntLength:155 TransferEncoding:[] Close:false Host:localhost:9090 Form:map[] PostForm:map[] MultipartForm:<nil> Trailer:
map[] RemoteAddr:[::1]:54684 RequestURI:/job TLS:<nil> Cancel:<nil> Response:<nil> ctx:0xc000412510}



*/

func logrusAccessLogShim(w io.Writer, params handlers.LogFormatterParams) {
	fmt.Printf("\n\nparams:%+v\n", params)
	fmt.Printf("\n\nparams.Request:%+v\n", params.Request)
	request := fmt.Sprintf("%s %s %s", params.Request.Method, params.Request.URL, params.Request.Proto)
	logger.Log.WithFields(logrus.Fields{
		"request": request,
		"status":  params.StatusCode,
		"size":    params.Size}).Info("access")

	/*
	   fmt.Println("\nSHITBALLS")

	*/
	//fmt.Fprintf(w, "{\"path\": \"%s\", \"status\": %d}", params.URL.Path, params.StatusCode)
}

/*
type log2LogrusWriter struct {
    logger *logrus.Logger
}

func (w *log2LogrusWriter) Write(b []byte) (int, error) {
 fmt.Println("HERE")
 n := len(b)
 if n > 0 && b[n-1] == '\n' {
  b = b[:n-1]
 }
 w.logger.Warning(string(b))
 return n, nil
}
*/

func loggingMiddleware(next http.Handler) http.Handler {
	//return handlers.CustomLoggingHandler(os.Stdout, next, shitballs)
	//logWriter := &log2LogrusWriter{logger.Log}
	//return handlers.CustomLoggingHandler(logWriter, next, shitballs)
	//return handlers.CustomLoggingHandler(logger.Log.Writer(), next, shitballs)
	return handlers.CustomLoggingHandler(ioutil.Discard, next, logrusAccessLogShim)
}
