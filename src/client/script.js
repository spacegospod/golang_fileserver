window.onload = function() {
	var USER_NAME = "";
	
	(function requestHomePage() {
		var req = new XMLHttpRequest();
        req.open("get", "/api/get/home/", true);
		req.onreadystatechange = function() {
    		if (req.readyState == 4) {
        		try {
					var pageData = req.responseText.split("&");
					USER_NAME = pageData[0];
					var dirInfo = JSON.parse(pageData[1]);
					updateTable(dirInfo);
				} catch (e) {
					// do nothing
				}
   			}
		}
		req.send();
	})();
	
	var uploadForm = document.getElementById('uploadForm');
	uploadForm.addEventListener("submit", function(e) {
		e.preventDefault();
		var fileSelect = document.getElementById('file');
		var file = fileSelect.files[0];
		
		var formData = new FormData();
		formData.append("file", file, file.name);
		
		var req = new XMLHttpRequest();
		req.open('post', "/api/post/upload/"+USER_NAME, true);
		req.onreadystatechange = function() {
			if (req.readyState == 4) {
				try {
					var dirInfo = JSON.parse(req.responseText);
					updateTable(dirInfo);
				} catch (e) {
					alert(req.responseText);
				}
			}
		}
		
		req.send(formData);
	});
	
	var mkdirButton = document.getElementById('mkdirButton');
	mkdirButton.addEventListener("click", function(e) {
		var dirname = document.getElementById('dirname').value;
		
		var req = new XMLHttpRequest();
		req.open('post', "/api/post/createdir/"+USER_NAME, true);
		req.onreadystatechange = function() {
			if (req.readyState == 4) {
				try {
					var dirInfo = JSON.parse(req.responseText);
					updateTable(dirInfo);
				} catch (e) {
					alert(req.responseText);
				}
			}
		}

		req.send(dirname);
	});
	
	var loginButton = document.getElementById('loginButton');
	loginButton.addEventListener("click", function(e) {	
			username = document.getElementById('username').value;
			password = document.getElementById('password').value;
			var req = new XMLHttpRequest();
            req.open("post", "/api/post/login/"+username, true);
			req.onreadystatechange = function() {
	    		if (req.readyState == 4 && req.responseText != "") {
	        		var resp = req.responseText.split('&');
					if (resp[0] == "failed") {
						alert(resp[1]);
					} else if (resp[0] == "success") {
						try {
							var dirInfo = JSON.parse(resp[1]);
							USER_NAME = username;
							updateTable(dirInfo);
						} catch (e) {
							alert(req.responseText);
						}
					}
	   			}
			}
			req.send('{"username":"'+username+'","password":"'+password+'"}');
	});
	
	var signupButton = document.getElementById('signupButton');
	signupButton.addEventListener("click", function(e) {	
			username = document.getElementById('username').value;
			password = document.getElementById('password').value;
			var req = new XMLHttpRequest();
            req.open("post", "/api/post/signup/"+username, true);
			req.onreadystatechange = function() {
	    		if (req.readyState == 4 && req.responseText != "") {
					alert(req.responseText);
				}
			}
			req.send('{"username":"'+username+'","password":"'+password+'"}');
	});
	
	function downloadHandler(e) {
		var fileName = e.target.parentNode.parentNode.childNodes[0].childNodes[0].wholeText;
		var req = new XMLHttpRequest();
        req.open("get", "/api/get/download/"+USER_NAME+"&"+fileName, true);
		req.onreadystatechange = function() {
    		if (req.readyState == 4) {
        		window.location = document.URL + "/api/get/download/"+USER_NAME+"&"+fileName;
   			}
		}
		req.send();
	}
		
	function deleteHandler(e) {
		var itemName = e.target.parentNode.parentNode.childNodes[0].childNodes[0].wholeText;
		var req = new XMLHttpRequest();
        req.open("delete", "/api/delete/"+USER_NAME+"&"+itemName, true);
		req.onreadystatechange = function() {
			if (req.readyState == 4) {
				try {
					var dirInfo = JSON.parse(req.responseText);
					updateTable(dirInfo);
				} catch (e) {
					alert(req.responseText);
				}
			}
		}
		req.send();
	}
	
	function openFolderHandler(e) {
		var folderName = e.target.parentNode.parentNode.childNodes[0].childNodes[0].wholeText;
		var req = new XMLHttpRequest();
        req.open("post", "/api/post/navigation/fwd&"+USER_NAME+"&"+folderName, true);
		req.onreadystatechange = function() {
			if (req.readyState == 4) {
				try {
					var dirInfo = JSON.parse(req.responseText);
					updateTable(dirInfo);
				} catch (e) {
					alert(req.responseText);
				}
			}
		}
		req.send();
	}
	
	function navigationBackHandler(e) {
		var req = new XMLHttpRequest();
        req.open("post", "/api/post/navigation/back&"+USER_NAME+"&", true);
		req.onreadystatechange = function() {
			if (req.readyState == 4) {
				try {
					var dirInfo = JSON.parse(req.responseText);
					updateTable(dirInfo);
				} catch (e) {
					alert(req.responseText);
				}
			}
		}
		req.send();
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
		var nameHeader = document.createElement('th');
		var backButton = document.createElement('img');
		backButton.src = "back_button.png";
		backButton.addEventListener("click", navigationBackHandler)
		nameHeader.appendChild(backButton);
		nameHeader.appendChild(document.createTextNode("Name"));
		var sizeHeader = document.createElement('th');
		sizeHeader.appendChild(document.createTextNode("Size(kb)"));
		headerRow.appendChild(nameHeader);
		headerRow.appendChild(sizeHeader);
		table.appendChild(headerRow);
		tableDiv.appendChild(table);
		
		if (dirInfo.Files != undefined) {
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
				downloadButton.addEventListener("click", downloadHandler);
				downloadColumn.appendChild(downloadButton);
				
				var deleteColumn = document.createElement('td');
				var deleteButton = document.createElement('img');
				deleteButton.src = "delete_button.png";
				deleteButton.addEventListener("click", deleteHandler);
				deleteColumn.appendChild(deleteButton);
				
				row.appendChild(fileNameColumn);
				row.appendChild(fileSizeColumn);
				row.appendChild(downloadColumn);
				row.appendChild(deleteColumn);
				
				table.appendChild(row);						
			});
		}

		if (dirInfo.Folders != undefined) {
			dirInfo.Folders.forEach(function(folderInfo) {
				var row = document.createElement('tr');
				
				var folderNameColumn = document.createElement('td');
				var folderName = document.createTextNode(folderInfo.Name);
				folderNameColumn.appendChild(folderName);
				
				var folderSizeColumn = document.createElement('td');
				var folderSize = document.createTextNode("--");
				folderSizeColumn.appendChild(folderSize);
				
				var openColumn = document.createElement('td');
				var openButton = document.createElement('img');
				openButton.src = "folder_button.png";
				openButton.addEventListener("click", openFolderHandler);
				openColumn.appendChild(openButton);
				
				var deleteColumn = document.createElement('td');
				var deleteButton = document.createElement('img');
				deleteButton.src = "delete_button.png";
				deleteButton.addEventListener("click", deleteHandler);
				deleteColumn.appendChild(deleteButton);
				
				row.appendChild(folderNameColumn);
				row.appendChild(folderSizeColumn);
				row.appendChild(openColumn);
				row.appendChild(deleteColumn);
				
				table.appendChild(row);			
			});
		}
	}
};
