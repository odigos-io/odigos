from django.http import HttpResponse

def index(request):
    return HttpResponse("Hello, this is a test app!")
