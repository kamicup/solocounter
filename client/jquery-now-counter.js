/**
 * The concurrent access counter service client library.
 */

(function($) {

    $.fn.asConcurrentAccessCounter = function(options){

        var defaults = {
            interval : 60,
            path : 'ping',
            server : 'http://solocounter.herokuapp.com/'
        };
        var setting = $.extend(defaults, options);

        var target = this;
        var endpoint = setting.server + setting.path;

        (function ping() {
            $.ajax({
                url: endpoint,
                success: function(data, textStatus, jqXHR) {
                    //console.log('success -> ' + textStatus);
                    target.text(data);
                },
                error: function(jqXHR, textStatus, errorThrown) {
                    //console.log('error -> ' + textStatus);
                }
            });

            setTimeout(ping, setting.interval * 1000);
        })();

        return (this);
    };

})(jQuery);
