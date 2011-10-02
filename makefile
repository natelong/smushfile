all: bin/smushfile

bin/smushfile: bin/smushfile.6
	6l -o bin/smushfile bin/smushfile.6
	rm bin/smushfile.6

bin/smushfile.6: smushfile.go
	mkdir bin
	6g -o bin/smushfile.6 smushfile.go

clean:
	rm -rf bin
