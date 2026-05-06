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
