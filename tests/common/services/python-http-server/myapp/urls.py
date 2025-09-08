# myapp/urls.py
from django.contrib import admin
from django.urls import path, include
from myapp_core import views

urlpatterns = [
    path('admin/', admin.site.urls),
    path('insert-random/', views.insert_random_row, name='insert-random'),
    path('fetch-all/', views.fetch_all_rows, name='fetch-all'),
    path('health/', include('health_check.urls')),
]
