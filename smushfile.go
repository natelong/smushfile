package main

import (
	"http"
	"log"
	"template"
	"io"
	"strings"
//	"json"
	"os"
)

var tmpls *template.Set

type Page struct {
	Title	string
	Data	string
}

type SmushRequest struct {
	Name	string
	URLs	[]string
	Result	string
}

func ReadWholeFile( f io.ReadCloser ) ( r string, err os.Error ) {
	defer f.Close()
	const NBUF = 512
    var buf [NBUF]byte
    for {
        switch nr, err := f.Read( buf[:] ); true {
        case nr < 0:
            log.Printf( "Error reading remote file: %v", err.String() )
			r = ""
			err = os.NewError( "Error reading remote file" )
			return
        case nr == 0: // EOF
            return
        case nr > 0:
            r += string( buf[ 0:nr ] )
        }
    }
	return
}

func SmushIndex( response http.ResponseWriter, request *http.Request ){
	var page = Page{
		Title: "Index",
	}

	err := tmpls.Execute( response, "index", page )
	if err != nil {
		log.Fatal( "Couldn't execute index template: ", err.String() )
	}
}

func SmushFiles( response http.ResponseWriter, request *http.Request ){

	// parse the form values from the body so we can interact with them
	err := request.ParseForm()
	if( err != nil ){
		log.Println( "Couldn't parse form values: ", err.String() )
		http.Error( response, "Couldn't parse form values", http.StatusInternalServerError )
		return
	}

	var source []string
	var exists bool

	// if there's a "source" param, iterate over its values and add them to a string for easy logging
	if source, exists = request.Form[ "source" ]; !exists {
		log.Println( "Couldn't parse form values: ", err.String() )
		http.Error( response, "Request missing 'source' param", http.StatusInternalServerError )
		return
	}

	sourceStrings := make( []string, len( source ) )
	i := 0

	for _, v := range source {
		if v != ""{
			sourceStrings[ i ] = v
			i++
		}
	}

	result := ""
	for _, v := range sourceStrings[ :i ]{
		r, err := http.Get( v )
		if( err != nil ){
			log.Printf( "Couldn't get %v: %v", v, err.String() )
			continue
		}
		log.Printf( "Successfully fetched: %v\n\t%v", r.Status, v )
		fileContents, _ := ReadWholeFile( r.Body )
		result += fileContents
	}

	response.Header().Add( "content-type", "text/plain" )
	io.WriteString( response, result )

	/*	
	response.Header().Add( "content-type", "application/json" )
	sReq := SmushRequest{
		"test",
		sourceStrings[ :i ],
		result,
	}

	sRes, err := json.Marshal( sReq )
	if err != nil{
		log.Println( "Couldn't marshal request to JSON: ", err.String() )
		http.Error( response, "Couldn't marshal request to JSON", http.StatusInternalServerError )
	}
	io.WriteString( response, string( sRes ) )
	*/
}

// Server static files, like CSS and JS directly
func StaticFile( response http.ResponseWriter, request *http.Request ){
	url := request.URL.Path	

	// remove the beginning slash so we don't look in the FS root for the file
	if url[ 0 ] == '/' {
		url = url[1:]
	}

	if strings.Contains( url, "../" ){
		log.Println( "Attempt to traverse outside of static folders." )
		http.Error( response, "Stop it.", http.StatusInternalServerError )
		return
	}

	// Log that a static file was served
	log.Print( "Serving ", url )
	
	// serve the file
	http.ServeFile( response, request, url )
}

func main(){
	if tmplSet, err := template.ParseSetFiles( "tmpl/header.html", "tmpl/footer.html", "tmpl/index.html" ); err != nil {
		log.Fatal( "Couldn't parse templates: ", err.String() )
	}else{
		tmpls = tmplSet
	}

	http.HandleFunc( "/out/", StaticFile )
	http.HandleFunc( "/css/", StaticFile )
	http.HandleFunc( "/js/", StaticFile )
	http.HandleFunc( "/smush", SmushFiles )
	http.HandleFunc( "/", SmushIndex )

	if err := http.ListenAndServe( ":12345", nil ); err != nil {
		log.Fatal( "ListenAndServe: ", err.String() )
	}
}
