$.ajax({url:"/authors/all", 
	success:function(data){
		console.log(data);
		for (i=0; i<data.length; i++){
			var html = "<li><a style='color:maroon' href="+data[i].Uri+">"+data[i].Name+"</a></li>";
			console.log(html);
			$(".link-list").append(html);
		}
	}
});
