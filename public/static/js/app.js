$.ajax({url:"/authors/all", 
	success:function(data){
		for (i=0; i<data.length; i++){
			var html = "<li><a href="+data[i].Uri+">"+data[i].Name+"</a></li>";
			$(".link-list").append(html);
		}
	}
});
