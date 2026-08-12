cmake_minimum_required(VERSION 3.19)
project(client LANGUAGES CXX)

find_package(Qt6 6.5 REQUIRED COMPONENTS Core Widgets Network)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

qt_standard_project_setup()

qt_add_executable(client
    src/main.cpp
    src/ui/mainwindow.cpp
    src/ui/mainwindow.h
    src/ui/mainwindow.ui
)

target_link_libraries(client
    PRIVATE
        Qt::Core
        Qt::Widgets
)

target_include_directories(client PRIVATE ${CMAKE_CURRENT_SOURCE_DIR}/src)
