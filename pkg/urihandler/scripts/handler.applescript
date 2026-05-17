on run
	-- Self-install: invoked when brew post_install does `open -a` on the
	-- bundle shipped under /opt/homebrew/.../share/. Copies the bundle into
	-- ~/Library/Application Support/apono-cli/ (the canonical location) and
	-- registers it with LaunchServices.
	set srcPath to POSIX path of (path to me)
	set homePath to POSIX path of (path to home folder)
	set destDir to homePath & "Library/Application Support/apono-cli"
	set destPath to destDir & "/Apono Connect.app"
	set lsr to "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"

	do shell script "mkdir -p " & quoted form of destDir
	do shell script "/usr/bin/ditto " & quoted form of srcPath & " " & quoted form of destPath
	do shell script quoted form of lsr & " -R " & quoted form of destPath
end run

on open location theURL
	set scriptPath to POSIX path of ((path to me as text) & "Contents:Resources:handler.sh")
	try
		do shell script "/bin/zsh -l " & quoted form of scriptPath & " " & quoted form of theURL
	on error errMsg number errNum
		if errNum is 64 then
			display dialog "Invalid launch URL. Please try again from the portal." with title "Apono" buttons {"OK"} default button "OK" with icon caution
		else
			display dialog ("Apono failed to launch:" & return & errMsg) with title "Apono" buttons {"OK"} default button "OK" with icon caution
		end if
	end try
end open location
