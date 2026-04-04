# CMake

Cross-platform build system generator that produces native build files (Makefiles, Ninja, Visual Studio projects) from declarative CMakeLists.txt configurations.

## Getting Started

```bash
# Check version
cmake --version

# Basic out-of-source build
mkdir build && cd build
cmake ..
cmake --build .

# Specify generator
cmake -G "Ninja" ..
cmake -G "Unix Makefiles" ..

# Set build type
cmake -DCMAKE_BUILD_TYPE=Release ..
cmake -DCMAKE_BUILD_TYPE=Debug ..
cmake -DCMAKE_BUILD_TYPE=RelWithDebInfo ..

# Install after building
cmake --install . --prefix /usr/local
```

## Minimal CMakeLists.txt

```cmake
cmake_minimum_required(VERSION 3.20)
project(myapp VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

add_executable(myapp src/main.cpp src/utils.cpp)
```

## Targets and Properties

```cmake
# Executable target
add_executable(myapp src/main.cpp)

# Static library
add_library(mylib STATIC src/lib.cpp)

# Shared library
add_library(mylib SHARED src/lib.cpp)

# Header-only (interface) library
add_library(myheaders INTERFACE)
target_include_directories(myheaders INTERFACE include/)

# Link libraries to targets (modern CMake)
target_link_libraries(myapp PRIVATE mylib)
target_link_libraries(myapp PUBLIC Threads::Threads)
target_link_libraries(myapp INTERFACE myheaders)

# Set include directories per-target
target_include_directories(mylib PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
    $<INSTALL_INTERFACE:include>
)

# Compile definitions per-target
target_compile_definitions(myapp PRIVATE DEBUG_MODE=1)

# Compile options per-target
target_compile_options(myapp PRIVATE -Wall -Wextra -Wpedantic)
```

## find_package

```cmake
# Find system-installed packages
find_package(OpenSSL REQUIRED)
target_link_libraries(myapp PRIVATE OpenSSL::SSL OpenSSL::Crypto)

find_package(Boost 1.70 REQUIRED COMPONENTS filesystem system)
target_link_libraries(myapp PRIVATE Boost::filesystem Boost::system)

find_package(Threads REQUIRED)
target_link_libraries(myapp PRIVATE Threads::Threads)

# Find package with optional components
find_package(Qt6 COMPONENTS Widgets Network QUIET)
if(Qt6_FOUND)
    target_link_libraries(myapp PRIVATE Qt6::Widgets Qt6::Network)
endif()

# Specify search paths
cmake -DCMAKE_PREFIX_PATH="/opt/mylibs;/usr/local" ..
```

## FetchContent (Dependency Download)

```cmake
include(FetchContent)

FetchContent_Declare(
    googletest
    GIT_REPOSITORY https://github.com/google/googletest.git
    GIT_TAG        v1.14.0
)

FetchContent_Declare(
    fmt
    GIT_REPOSITORY https://github.com/fmtlib/fmt.git
    GIT_TAG        10.2.1
)

# Download and make available
FetchContent_MakeAvailable(googletest fmt)

target_link_libraries(myapp PRIVATE fmt::fmt)
target_link_libraries(tests PRIVATE GTest::gtest_main)
```

## Generator Expressions

```cmake
# Conditional based on build config
target_compile_definitions(myapp PRIVATE
    $<$<CONFIG:Debug>:DEBUG_BUILD>
    $<$<CONFIG:Release>:NDEBUG>
)

# Conditional based on compiler
target_compile_options(myapp PRIVATE
    $<$<CXX_COMPILER_ID:GNU>:-Wall -Wextra>
    $<$<CXX_COMPILER_ID:MSVC>:/W4>
)

# Install vs build interface
target_include_directories(mylib PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
    $<INSTALL_INTERFACE:include>
)

# Boolean logic
$<AND:$<CONFIG:Debug>,$<PLATFORM_ID:Linux>>
$<OR:$<CXX_COMPILER_ID:GNU>,$<CXX_COMPILER_ID:Clang>>
$<NOT:$<BOOL:${SOME_VAR}>>
```

## Install Rules

```cmake
include(GNUInstallDirs)

install(TARGETS myapp mylib
    EXPORT myproject-targets
    RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR}
    LIBRARY DESTINATION ${CMAKE_INSTALL_LIBDIR}
    ARCHIVE DESTINATION ${CMAKE_INSTALL_LIBDIR}
)

install(DIRECTORY include/
    DESTINATION ${CMAKE_INSTALL_INCLUDEDIR}
)

install(FILES LICENSE README.md
    DESTINATION ${CMAKE_INSTALL_DOCDIR}
)

# Export targets for downstream find_package
install(EXPORT myproject-targets
    FILE myproject-targets.cmake
    NAMESPACE myproject::
    DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/myproject
)
```

## CPack (Packaging)

```cmake
set(CPACK_PACKAGE_NAME "myapp")
set(CPACK_PACKAGE_VERSION ${PROJECT_VERSION})
set(CPACK_PACKAGE_DESCRIPTION_SUMMARY "My application")
set(CPACK_PACKAGE_CONTACT "dev@example.com")

# DEB package
set(CPACK_DEBIAN_PACKAGE_DEPENDS "libssl-dev (>= 1.1)")

# RPM package
set(CPACK_RPM_PACKAGE_REQUIRES "openssl-devel >= 1.1")

include(CPack)
```

```bash
# Generate packages
cd build
cpack -G DEB
cpack -G RPM
cpack -G TGZ
cpack -G ZIP
```

## CMake Presets

```json
{
    "version": 6,
    "configurePresets": [
        {
            "name": "debug",
            "generator": "Ninja",
            "binaryDir": "${sourceDir}/build/debug",
            "cacheVariables": {
                "CMAKE_BUILD_TYPE": "Debug",
                "CMAKE_EXPORT_COMPILE_COMMANDS": "ON"
            }
        },
        {
            "name": "release",
            "generator": "Ninja",
            "binaryDir": "${sourceDir}/build/release",
            "cacheVariables": {
                "CMAKE_BUILD_TYPE": "Release"
            }
        }
    ],
    "buildPresets": [
        { "name": "debug", "configurePreset": "debug" },
        { "name": "release", "configurePreset": "release" }
    ]
}
```

```bash
# Use presets
cmake --preset debug
cmake --build --preset debug
ctest --preset debug
```

## Testing with CTest

```cmake
enable_testing()

add_executable(tests test/test_main.cpp)
target_link_libraries(tests PRIVATE GTest::gtest_main mylib)

include(GoogleTest)
gtest_discover_tests(tests)

# Or manual test registration
add_test(NAME unit_tests COMMAND tests)
add_test(NAME integration COMMAND ${CMAKE_SOURCE_DIR}/test/run_integration.sh)
```

```bash
cd build
ctest --output-on-failure
ctest -j$(nproc)               # parallel
ctest -R "unit"                 # filter by regex
ctest --test-dir build/debug
```

## Custom Commands and Targets

```cmake
# Custom command (file generation)
add_custom_command(
    OUTPUT ${CMAKE_BINARY_DIR}/generated.h
    COMMAND python3 ${CMAKE_SOURCE_DIR}/scripts/codegen.py
    DEPENDS ${CMAKE_SOURCE_DIR}/scripts/codegen.py
    COMMENT "Generating header"
)

# Custom target (always runs)
add_custom_target(format
    COMMAND clang-format -i ${ALL_SOURCES}
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
    COMMENT "Running clang-format"
)
```

## Tips

- Always use target-based commands (`target_link_libraries`, `target_include_directories`) instead of directory-based ones (`link_libraries`, `include_directories`)
- Set `CMAKE_EXPORT_COMPILE_COMMANDS ON` for IDE and clang-tidy integration
- Use `PRIVATE`/`PUBLIC`/`INTERFACE` visibility correctly: PRIVATE for implementation, PUBLIC for both, INTERFACE for header-only
- Prefer `FetchContent` over `ExternalProject_Add` for dependencies used at configure time
- Use `cmake --build . -j$(nproc)` instead of calling `make` directly for generator portability
- Always do out-of-source builds (never `cmake .` in the source directory)
- Use `CMakePresets.json` for reproducible builds across developers and CI
- Set `CMAKE_CXX_STANDARD` instead of manually adding `-std=c++17` flags
- Use generator expressions for config-dependent settings instead of `if(CMAKE_BUILD_TYPE)`
- Enable `-Werror` in CI but not in development presets to avoid blocking iteration
- Use `find_package` with imported targets (e.g., `OpenSSL::SSL`) rather than raw variables
- Run `cmake --graphviz=deps.dot .` to visualize target dependency graphs

## See Also

- Ninja build system
- Conan and vcpkg package managers
- Meson build system
- Autotools (legacy)
- GCC and Clang compilers

## References

- [CMake Documentation](https://cmake.org/cmake/help/latest/)
- [CMake Tutorial](https://cmake.org/cmake/help/latest/guide/tutorial/index.html)
- [Modern CMake (Henry Schreiner)](https://cliutils.gitlab.io/modern-cmake/)
- [Professional CMake (Craig Scott)](https://crascit.com/professional-cmake/)
- [CMake FetchContent Module](https://cmake.org/cmake/help/latest/module/FetchContent.html)
- [CMake Generator Expressions](https://cmake.org/cmake/help/latest/manual/cmake-generator-expressions.7.html)
