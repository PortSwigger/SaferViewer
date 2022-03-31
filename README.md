# SaferViewer

This utility is a Mac application which you can 'drag and drop' files onto, as a safer way to open them avoiding using applications on your host machine.  It is intended to be used by end users to preview documents, such as those attached to emails.  

When a document is dragged onto the app, the very first time, the user will be directed to 
allow the application access to the user google drive via a standard google login dialog.  Once
approved, the contents of the "responsefile.html" will be displayed.

Going forwards, when a user drags a document onto the icon the application will upload it in the
background to the users google drive into a predefined folder (SaferViewer) and then, open the 
users web browser to view the document using the Google Docs 'preview' view.  At this point, 
the user can review the document, and if they wish to print or do more with it, can click the 
icon on the top right to be presented with more options.  From this latter screen, they can then
choose to edit the document in normal Google Docs type interface.  Alternatively they could simply
'double click' the original source to open the document in the default editor on the host machine.

## Building

To start, you'll need to create a "client_secret.json" file, which is the output
of following the process documented at https://developers.google.com/workspace/guides/get-started

In terse summary, you'll need to enable the GoogleDrive API in your organisations
account, and set up OAuth 2.0 for authorization.  After creating the OAuth stuff, you'll
download a JSON file which is the authentication data for this application, named as above. At
build time, this file content is embedded into the binary.  There's an obvious residual
risk as this data is sensitive, which you may wish to remedy somehow.

Once built, you need to create a 'wrapper' application for the utility, so that
the Events framework is supported.  The utility itself just requires the name
of the input file as argv[1].

The easiest way to do this is to create an app using the 'Automator' command on
MacOS.  Open the App, then select 'Application' as the 'type for your document'.
Under the 'Library' on the left pane, select 'Utilities' and 'Run Shell Script',
drag this to the right Workflow pane.  You can leave the preselected 'zsh' as the interpreter, it 
makes no difference.  Change 'pass input' to be 'as arguments, then change the contents to be 
something like this:

`/Applications/WhateverYouCallIt.app/Contents/MacOS/SaferViewer "$@"`

`exit 0`

Then, when you save the file, it will create a universal binary.  The next thing 
to do is to copy the golang binary to the above referenced file, and (optionally)
change the icon and re-sign the application (left as an exercise to the reader).


