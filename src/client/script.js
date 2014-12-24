window.onload = function() {	
/*
	var uploadButton = document.getElementById('uploadButton');
	uploadButton.addEventListener("click", function(e) {
            var file = document.getElementById('fileSelector');

            var formData = new FormData();
			console.log(file.files);
            formData.append("upload", file.files[0]);

			var req = new XMLHttpRequest();
            req.open("put", "/api/", true);
            req.setRequestHeader("Content-Type", "multipart/form-data");
            req.send(formData);
	});
*/
	var downloadButton = document.getElementById('downloadButton');
	downloadButton.addEventListener("click", function(e) {
			var fileName = document.getElementById('downloadName').value;
			var req = new XMLHttpRequest();
            req.open("get", "/api/download/"+fileName, true);
            req.send();
			req.onreadystatechange = function() {
	    		if (req.readyState == 4) {
	        		window.location = document.URL + "/api/download/" + fileName;
	   			}
			}
	});
	
	var loginButton = document.getElementById('loginButton');
	loginButton.addEventListener("click", function(e) {	
			username = document.getElementById('username').value;
			password = document.getElementById('password').value;
			var req = new XMLHttpRequest();
            req.open("post", "/api/post/login/"+username, true);
            req.send('{"username":"'+username+'","password":"'+password+'"}');
	});
};
