$(function() {
				   
	var Posts = Backbone.Collection.extend({
		url: "/feeds/all"
	});
	// model
	var Post = Backbone.Model.extend({
	});

	var p = new Posts;
	p.fetch();
});
