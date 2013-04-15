$.ajax({url:"/authors/all", 
	success:function(data){
		for (i=0; i<data.length; i++){
			var html = "<li><a style='color:maroon' href="+data[i].Uri+">"+data[i].Name+"</a></li>";
			$(".link-list").append(html);
		}
	}
});
