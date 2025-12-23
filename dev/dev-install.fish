#! /usr/bin/env fish

function install-if-missing --argument-names cmd pkg_name
    echo "Checking for $cmd..."
    if not type -q $cmd
        echo "...not found"
        echo "Checking for brew..."
        if not type -q brew
            echo "...not found"
            echo "Installing brew..."
            /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
        end
        echo "Checking for brew..."
        if not type -q brew
            echo "...not found"
            echo "Unable to continue installing dev tools: brew failed to install."
            exit 1
        end
        echo "...found"

        if test -z $pkg_name
            set pkg_name $cmd
        end
        echo "Installing $pkg_name..."
        brew install $pkg_name

        echo "Checking for $cmd..."
        if not type -q $cmd
            echo "...not found"
            echo "Unable to continue installing dev tools: $cmd failed to install."
            exit 1
        end
    end
    echo "...found"
end

install-if-missing sed gnu-sed
install-if-missing awk
install-if-missing ag
install-if-missing go
install-if-missing golangci-lint
install-if-missing mage
go install go.uber.org/nilaway/cmd/nilaway@latest
go install golang.org/x/tools/cmd/deadcode@latest
