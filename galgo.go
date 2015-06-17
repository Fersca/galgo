package galgo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"

	//"github.com/yvasiyarov/gorelic"
)

//Cliente http para relizar las conexiones contra la API
var client *http.Client

//Agente de new relic
//var agent *gorelic.Agent

//URLBase para la llamada a los recursos
var BaseURL string

//variable que hostea el entorno
var environment map[string]bool = make(map[string]bool)

//Headers solo para testing
var HeadersTesteo http.Header

//Start - Start the application
func Start() {

	//setea todos los cores para funcionar
	coresSetup()

	//Setea la url Base
	urlBaseSetup()

	//Setea el agente de new relic
	//newRelicSetup()

	//crea un cliente http para conectarse contra la api
	httpClientSetUp()

	//crea el webserver que atiende los pedidos
	webserverSetUp()

	//Imprime el cartel de bienvenida
	welcomeScreen()

	//Imprime la pantalla de bienvenida
	welcomeScreen()

}

//Imprime la pantalla de bienvenida
func welcomeScreen() {

	if environment["production"] {
		println("Production Environment")
	} else {
		println("Developent Environment")
	}
}

//Setea los cores para los threads
func coresSetup() {
	coreNum := runtime.NumCPU()
	runtime.GOMAXPROCS(coreNum)
}

//Setea la url base
func urlBaseSetup() {

	environment["development"] = false
	environment["test"] = false
	environment["production"] = false

	if os.Getenv("DATACENTER") == "" {
		BaseURL = "https://api.mercadolibre.com/"
		environment["development"] = true
	} else {
		BaseURL = "http://internal.mercadolibre.com/"
		environment["production"] = true
	}
}

type Context struct {
	Response http.ResponseWriter
	Request  *http.Request
	Params   map[string]string
}

func (c *Context) Render(content string) {
	c.Response.Write([]byte(processJSONP(content, c.Params, c.Response.Header())))
}

func (c *Context) RenderJSON(content map[string]interface{}) {
	marshaled, _ := json.Marshal(content)
	c.Render(string(marshaled))
}
func (c *Context) SetStatusCode(code int) {
	c.Response.WriteHeader(code)
}
func (c *Context) SetHeader(key string, value string) {
	c.Response.Header().Add("Cache-Control", "private, max-age=1800")
}

type mapRout struct {
	handler (func(c *Context))
	verb    string
}

var mapping2 map[string]*mapRout = make(map[string]*mapRout)

//AddController add a controller to process the request
func AddController(verb string, url string, handler2 func(c *Context)) {
	m1 := new(mapRout)
	m1.handler = handler2
	m1.verb = verb
	mapping2[url] = m1
}

func preFilter(w http.ResponseWriter, req *http.Request) {

	//Get the mapping route
	mapRout := mapping2[req.URL.Path]
	if mapRout == nil {
		mapRout = mapping2["default"]
		if mapRout == nil {
			w.WriteHeader(404)
			w.Write([]byte("Not Found"))
			return
		}
	}

	//Create the context
	c := new(Context)
	c.Request = req
	c.Response = w
	c.Params = getParams(req)

	//Obtiene el mapa de headers
	headerMap := w.Header()

	//Agrega el header de que se devuelve un json
	headerMap.Add("Content-Type", "application/json;charset=UTF-8")

	//check if the verb is available for the url
	switch req.Method {

	case mapRout.verb:
		mapRout.handler(c)

	default:
		//devuelve un not Not Allowed
		w.WriteHeader(405)
	}

}

//Crea el webserver para atajar los llamados al multiget
func webserverSetUp() {
	//http.HandleFunc("/", agent.WrapHTTPHandlerFunc(processRequest))
	for key := range mapping2 {
		http.HandleFunc(key, http.HandlerFunc(preFilter))
	}

	err := http.ListenAndServe("0.0.0.0:8080", nil)
	if err != nil {
		println("Multiget ListenAndServe Error", err)
	}
}

//Agrega los datos para jsonP si es que se llama con callback
func processJSONP(resultado string, params map[string]string, headerMap http.Header) string {

	//Verifica si el llamado en en jsonP
	if ContainsKey(params, "callback") {

		//Quita el valor anterior del header
		headerMap.Del("Content-Type")

		//Cambia el header del content type
		headerMap.Add("Content-Type", "text/javascript;charset=UTF-8")

		//genera el resultado para jsonP
		return params["callback"] + "([200, {\"Content-Type\":\"application/json;charset=UTF-8\"}," + resultado + "]);"

	}

	return resultado

}

//Crea el cliente de http para conectarse a las apis
func httpClientSetUp() {
	//crea un transport para la conxion
	tr := &http.Transport{
		DisableCompression:  false,
		DisableKeepAlives:   false,
		MaxIdleConnsPerHost: 100,
	}

	//crea un cliente http para conectarse contra la api
	client = &http.Client{Transport: tr}
}

//Setea el agente de new relic
/*
func newRelicSetup() {
	agent = gorelic.NewAgent()
	agent.NewrelicName = "MultigetAPI"
	agent.Verbose = true
	agent.CollectHTTPStat = true
	agent.NewrelicLicense = "db29d78bd11c35aa77ba7dfe9fb2e6ab2b550c2e"
	agent.Run()
}
*/
//Funciones de impresión

//funcion que simula el println de groovy
func println(a ...interface{}) {
	for id := range a {
		fmt.Print(a[id])
	}
}

//Funciones para manejo de HTTP

//Obtiene los parametros del request y devuelve un mapa con los mismos
func getParams(req *http.Request) map[string]string {

	var mapa = make(map[string]string)

	firstCut := strings.Split(req.URL.RequestURI(), "?")

	if len(firstCut) > 1 {
		values := strings.Split(firstCut[1], "&")

		for _, value := range values {
			keyValue := strings.Split(value, "=")
			mapa[keyValue[0]] = keyValue[1]
		}
	}
	return mapa
}

//chequea si el mapa contiene el string como key
func ContainsKey(mapa map[string]string, palabra string) bool {
	for key := range mapa {
		if key == palabra {
			return true
		}
	}
	return false
}

//chequea si el array contiene el string
func Contains(array []string, palabra string) bool {
	for _, value := range array {
		if value == palabra {
			return true
		}
	}
	return false
}

func ProcessGetTest(urlString string, headers map[string]string) *httptest.ResponseRecorder {

	//create the request
	request, _ := http.NewRequest("GET", urlString, nil)

	//copy the headers
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response := httptest.NewRecorder()

	//execute the request
	preFilter(response, request)

	return response
}

func ProcessDeleteTest(urlString string) *httptest.ResponseRecorder {

	//create the post request
	request, _ := http.NewRequest("DELETE", urlString, nil)
	response := httptest.NewRecorder()

	//execute the request
	preFilter(response, request)

	return response
}

//http GET pulido
func Get(urlString string, headers map[string]string) (string, int, error) {

	//crea el request a la api
	req, err := http.NewRequest("GET", urlString, nil)

	if err != nil {
		println("Error in http.NewRequest:", err)
	}

	//Setea los headers para llamar a las apis
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Connection", "Keep-Alive")

	//pasa los headers que vinieron al nuevo request
	if headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

	//solamente para poder hacer un test y ver si se setean bien los headers
	if !environment["production"] {
		HeadersTesteo = req.Header
	}

	//realiza el request mediante el cliente
	resp, err2 := client.Do(req)
	if err2 != nil {
		return "", 0, err2
	}

	code := resp.StatusCode
	//leo el body de la respuesta
	body, err3 := ioutil.ReadAll(resp.Body)
	if err3 != nil {
		return "", code, err2
	}

	//difiere el cierr del body
	defer func() {
		//chequeo si no es nil porque en caso de error al abrirlo falla esta respuesta.
		if resp != nil {
			resp.Body.Close()
		}
	}()

	return string(body), code, nil

}

/////////////// Testing Functions /////////////////

//CheckContent check the body content of a response
func CheckContent(t *testing.T, response *httptest.ResponseRecorder, content string) {

	body, _ := ioutil.ReadAll(response.Body)
	if string(body) != content {
		t.Fatalf("Non-expected content %s, expected %s", string(body), content)
	}
}

//CheckStatus Check the status code
func CheckStatus(t *testing.T, response *httptest.ResponseRecorder, expected int) {
	if response.Code != expected {
		t.Fatalf("Non-expected status code %v :\n\tbody: %v", expected, response.Code)
	}
}

//CheckHeader Check the specific header value
func CheckHeader(t *testing.T, response *httptest.ResponseRecorder, header string, value string) {
	if response.Header().Get(header) != value {
		t.Fatalf("Header: %s, get:%s expected:%s", header, response.Header().Get(header), value)
	}
}

//ConvertJSONToMap Convert a Json string to a map
func JSONResponseToMap(response *httptest.ResponseRecorder) []interface{} {

	body, _ := ioutil.ReadAll(response.Body)
	valor := string(body)

	//Create the Json element
	d := json.NewDecoder(strings.NewReader(valor))
	d.UseNumber()
	var f interface{}
	err := d.Decode(&f)

	if err != nil {
		return nil
	}

	return f.([]interface{})

}

//JSONElement Returns the the element in the position
func JSONResponseElementInArray(response *httptest.ResponseRecorder, element int) map[string]interface{} {
	mapa := JSONResponseToMap(response)
	return mapa[element].(map[string]interface{})
}
