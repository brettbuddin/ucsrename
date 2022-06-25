ucsrename renames files using [Universal Category System
(UCS)](https://universalcategorysystem.com) filename pattern.

Usage:

	ucsrename [-y] filename.wav

The program asks a series of questions to build a filename that conforms to UCS
standards. The source file's file extension is carried forward to the new file.
Here's the layout of the filename that it produces:

	CatID_FXName_CreatorID_SourceID_UserData.Extention

CatID, FXName, CreatorID and SourceID are required fields. The UserData field is
optional and can be to specify information not captured by the UCS standard.

[fzf](https://github.com/junegunn/fzf) is required to provide a helpful,
filterable, list of category IDs.

The UCS project has a great video outlining the filename structure:
https://www.youtube.com/watch?v=0s3ioIbNXSM
