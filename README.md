# SaferViewer

This utility is a Mac application which you can 'drag and drop' files onto, as a safer way to
open them avoiding using applications on your host machine.  It is intended for end users
to preview documents, such as those attached to emails.  Note, this is only of use to you if
you are already hooked into using Google Docs for your organisation.  Of course, you could
re-implement the concept to suite your use case.

Most of the attacks we see via email are usually attachments, such as spurious CSV files or
document attacks which are expect to be loaded via Microsoft Word/Pages or Excel/Numbers by the
victim user.  

While most office applications prompt users with something like "Are you completely,
absolutely, infalliably certain you want to open this document?" people make mistakes, or are
under pressure and could well click through the warnings.  

The concept here is that we simply open the document via a safer mechanism, namely we'll open it in the users web browser using all the cleverness that google has implemented in their google drive/google docs, which is to say things like the local filesystem won't be available to the document, nor will macros execute, etc.  

This removes a suprising amount of exploit vectors...  A comedic side effect is that it's often 
fast to open the document via this method than for a user to cold load their document in their
office suite

When a document is dragged onto the app, the very first time, the user will be directed to 
allow the application access to the user google drive via a standard google login dialog.  Once
approved, the contents of the "responsefile.html" will be displayed.  Note: this file is embedded 
at build time.

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

To actually build, the included `Makefile' has default target for building a Darwin/amd64 target,
which should support both the intel and arm architectures, assuning you've already got Rosetta 
installed.  It's rather unlikely users on M1's have got this far without requiring it, but ymmv.  

The `make install' target should be used once you've created the wrapper binary, as described
below.  You will need to customise the Makefile to change file locations if you deviate from
naming the app SaferViewer, or the app install location.

## Wrapper

Once built, you need to create a 'wrapper' application for the utility, so that
the Apple Events framework is supported.  The utility itself just requires the name
of the input file as argv[1].

The easiest way to do this is to create an app using the 'Automator' command on
MacOS.  Open the App, then select 'Application' as the 'type for your document'.
Under the 'Library' on the left pane, select 'Utilities' and 'Run Shell Script',
drag this to the right Workflow pane.  You can leave the preselected 'zsh' as the interpreter, it 
makes no difference.  Change 'pass input' to be 'as arguments', then change the contents to be 
something like this:

`/Applications/SaferViewer.app/Contents/MacOS/SaferViewer "$@"`

`exit 0`

(Note: we exit 0 as users will be confused by 'raw' error messages.  Debugging information is
 written to logfile variable definition in the code)

Then, when you save the file, it will create a universal binary.  

## Adding the golang binary to the .app

The next thing to do is to copy the golang binary to the above referenced file, the Makefile 
`install' target will do this.  

Optionally, change the icon and re-sign the application (left as an exercise to the reader).