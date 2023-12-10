#! /usr/bin/env fish

# Workaround to fuzz all the fuzz tests we can write, roughly stolen from https://github.com/golang/go/issues/46312#issuecomment-1153345129
# Additionally, the use of ag to match a single part of the function was informed by https://github.com/ggreer/the_silver_searcher/issues/400#issuecomment-418454903

set files (ag 'func Fuzz' -G '.*test.go' -l; or true)

for file in $files
    echo Fuzz tests found in $file
    set funcs (ag -o '^func \KFuzz.*(?=\()' $file)
    for func in $funcs
        echo Fuzzing $func
        go test ./(dirname $file) -fuzz $func -fuzztime 1000x
    end
end
