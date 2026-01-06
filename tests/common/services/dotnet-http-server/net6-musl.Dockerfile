FROM mcr.microsoft.com/dotnet/sdk:6.0 AS build
WORKDIR /src
COPY . .
ENV USE_DOTNET6=true
RUN dotnet restore
RUN dotnet publish -c Release -o /app

FROM mcr.microsoft.com/dotnet/aspnet:6.0-alpine
RUN apk add --no-cache icu-libs libintl
WORKDIR /app
COPY --from=build /app .
ENV DOTNET_SYSTEM_GLOBALIZATION_INVARIANT=false
EXPOSE 8080
ENTRYPOINT ["dotnet", "dotnet-http-server.dll"]
