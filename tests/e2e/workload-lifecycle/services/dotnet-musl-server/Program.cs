using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Hosting;
using Microsoft.AspNetCore.Http;
using Microsoft.Extensions.Logging;
using System;


namespace LegacyWebHostTest
{
    public class Program
    {
        public static void Main(string[] args)
        {
            var host = new WebHostBuilder()
                // Configure Kestrel, etc.
                .UseKestrel(options =>
                {
                    options.ListenAnyIP(8080);
                })

                // Logging
                .ConfigureLogging(logging =>
                {
                    logging.ClearProviders();
                    logging.AddConsole();
                    // Typically with 2.x hosting, AddConsole()
                    // sets up the console logger, including logger for WebHost
                })

                // Minimal pipeline
                .Configure(app =>
                {
                    app.Run(async context =>
                    {
                        var logger = context
                            .RequestServices
                            .GetService(typeof(ILogger<Program>))
                            as ILogger<Program>;

                        logger?.LogInformation("Handling request on path {path}", context.Request.Path);

                        await context.Response.WriteAsync("Hello from a legacy webhost on .NET 6\n");
                    });
                })

                // Build the host
                .Build();

            // Run
            host.Run();
        }
    }
}
