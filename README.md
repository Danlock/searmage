# searmage

Searmage is a CLI tool for searching through your local images. It can watch a folder and build up an index for quickly searching through images later. Powered by Tesseract and SQLite.

# example

Check out this repo and run ` make build `

` ./bin/searmage -dir /some/folder/with/images -db /tmp/searmage.sqlite3 `

This will parse images in the given directory for any text, and then store the text within the sqlite database.

` ./bin/searmage -search '%the meaning of life%' -db /tmp/searmage.sqlite3 `

This will search the previously parsed image text and return with the path of matching images.

` ./bin/searmage -help `

Will expose further flags, outlined at cfg/args.go
