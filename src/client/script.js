window.onload = function() {
	var uploadForm = document.getElementById('uploadForm');
	uploadForm.addEventListener("submit", function(e) {
		e.preventDefault();
	});
	
	var loginButton = document.getElementById('loginButton');
	loginButton.addEventListener("click", function(e) {	
			username = document.getElementById('username').value;
			password = document.getElementById('password').value;
			var req = new XMLHttpRequest();
            req.open("post", "/api/post/login/"+username, true);
            req.send('{"username":"'+username+'","password":"'+password+'"}');
			req.onreadystatechange = function() {
	    		if (req.readyState == 4 && req.responseText != "") {
	        		var resp = req.responseText.split('&');
					if (resp[0] == "failed") {
						alert(resp[1]);
					} else if (resp[0] == "success") {
						var dirInfo = JSON.parse(resp[1]);
						updateTable(dirInfo);
					}
	   			}
			}
	});
	
	var signupButton = document.getElementById('signupButton');
	signupButton.addEventListener("click", function(e) {	
			username = document.getElementById('username').value;
			password = document.getElementById('password').value;
			var req = new XMLHttpRequest();
            req.open("post", "/api/post/signup/"+username, true);
            req.send('{"username":"'+username+'","password":"'+password+'"}');
			req.onreadystatechange = function() {
	    		if (req.readyState == 4 && req.responseText != "") {
					alert(req.responseText);
				}
			}
	});
	
	function downloadHandler(e) {
		var fileName = e.target.parentNode.parentNode.childNodes[0].childNodes[0].wholeText;
		var req = new XMLHttpRequest();
        req.open("get", "/api/download/"+fileName, true);
        req.send();
		req.onreadystatechange = function() {
    		if (req.readyState == 4) {
        		window.location = document.URL + "/api/download/" + fileName;
   			}
		}
	}
		
	function deleteHandler(e) {
		var fileName = e.target.parentNode.parentNode.childNodes[0].childNodes[0].wholeText;
		var req = new XMLHttpRequest();
        req.open("delete", "/api/delete/"+fileName, true);
        req.send();
		req.onreadystatechange = function() {
			if (req.readyState == 4) {
				var dirInfo = JSON.parse(req.responseText);
				updateTable(dirInfo);
			}
		}
	}
	
	function updateTable(dirInfo) {
		var table = document.getElementById('table');
		// clear existing table data
		var tableDiv = document.getElementById('tablediv');
		tableDiv.removeChild(table);
		
		// generate new table
		table = document.createElement('table');
		table.id = "table";
		table.border = "1"
		
		var headerRow = document.createElement('tr');
		var filenameHeader = document.createElement('th');
		filenameHeader.appendChild(document.createTextNode("Filename"));
		var filesizeHeader = document.createElement('th');
		filesizeHeader.appendChild(document.createTextNode("Size(kb)"));
		headerRow.appendChild(filenameHeader);
		headerRow.appendChild(filesizeHeader);
		table.appendChild(headerRow);
		tableDiv.appendChild(table);
		
		dirInfo.Files.forEach(function(fileInfo) {
			var row = document.createElement('tr');
			
			var fileNameColumn = document.createElement('td');
			var fileName = document.createTextNode(fileInfo.Name);
			fileNameColumn.appendChild(fileName);
			
			var fileSizeColumn = document.createElement('td');
			var fileSize = document.createTextNode(Math.floor(fileInfo.Size / 1000));
			fileSizeColumn.appendChild(fileSize);
			
			var downloadColumn = document.createElement('td');
			var downloadButton = document.createElement('img');
			downloadButton.src = "download_button.png";
			downloadColumn.appendChild(downloadButton);
			downloadButton.addEventListener("click", downloadHandler);
			
			var deleteColumn = document.createElement('td');
			var deleteButton = document.createElement('img');
			deleteButton.src = "delete_button.png";
			deleteColumn.appendChild(deleteButton);
			deleteButton.addEventListener("click", deleteHandler);
			
			row.appendChild(fileNameColumn);
			row.appendChild(fileSizeColumn);
			row.appendChild(downloadColumn);
			row.appendChild(deleteColumn);
			
			table.appendChild(row);						
		});
	}
};
