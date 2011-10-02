package main

import (
	"http"
	"log"
	"template"
	"io"
	"strings"
	"os"
	"url"
	"time"
	"strconv"
)

var tmpls *template.Set

const compilerURL = "http://closure-compiler.appspot.com/compile"

type Page struct {
	Title	string
	Data	string
}

func ReadWholeFile( f io.ReadCloser ) ( r string, err os.Error ) {
	defer f.Close()
	const NBUF = 512
    var buf [ NBUF ]byte
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

func RequireParams( request *http.Request, params []string ) ( success bool ){
	err := request.ParseForm()
	if err != nil {
		return false
	}

	for _, v := range params{
		if thisParam, exists := request.Form[ v ]; exists{
			hasGoodValue := false;
			for _, paramValue := range thisParam{
				if len( paramValue ) > 0{
					hasGoodValue = true
					break;
				}
			}
			if !hasGoodValue{
				return false
			}
		}else{
			return false
		}
	}

	return true
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
	var requiredParams = []string{ "name", "source" }

	if request.Method != "POST" {
		http.Error( response, "Must post from main Smush form", http.StatusInternalServerError )
		return
	}

	if !RequireParams( request, requiredParams ){
		http.Error( response, "Missing required params", http.StatusInternalServerError )
		return
	}

	var source []string

	source = request.Form[ "source" ]
	sourceStrings := make( []string, len( source ) )
	i := 0

	// move the source urls into a new array so we can preserve the order, but skip blank entries
	for _, v := range source {
		if v != ""{
			sourceStrings[ i ] = v
			i++
		}
	}

	compileParamString := "compilation_level=SIMPLE_OPTIMIZATIONS"
	compileParamString += "&output_format=text"
	compileParamString += "&output_info=compiled_code"

	for _, v := range sourceStrings[ :i ]{
		compileParamString += "&code_url=" + v
	}

	compileParams, _ := url.ParseQuery( compileParamString )
	compileResponse, _ := http.PostForm( compilerURL, compileParams )

	result, _ := ReadWholeFile( compileResponse.Body )
	if len( result ) == 1 {
		http.Error( response, "Something went wrong with the Closure Compiler", http.StatusInternalServerError )
		return
	}

	currentTime := time.LocalTime().Nanoseconds()

	dirName := "out/" + strconv.Itoa64( currentTime )
	fileName := dirName + "/" + request.Form[ "name" ][ 0 ] + ".min.js"
	if err := os.Mkdir( dirName, uint32( 0777 ) ); err != nil{
		http.Error( response, "Couldn't create output folder", http.StatusInternalServerError )
		return
	}
	outputFile, err := os.Create( fileName )
	if err != nil{
		http.Error( response, "Couldn't create output file", http.StatusInternalServerError )
		return
	}
	_, err = outputFile.WriteString( result )
	if err != nil{
		http.Error( response, "Couldn't write output to file", http.StatusInternalServerError )
		return
	}

	http.Redirect( response, request, "/" + fileName, http.StatusFound )
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
	log.Print( "Serving static file ", url )
	
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
