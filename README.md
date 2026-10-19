# PhotoPi

PhotoPi turns a computer with a camera and a screen into a photo booth.

## Quick start

On Linux:

```sh
curl -fsSLO https://raw.githubusercontent.com/barthjs/photopi/main/scripts/install.sh
bash install.sh --user
```

On Windows, in PowerShell 7:

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/barthjs/photopi/main/scripts/install.ps1 -OutFile install.ps1
pwsh -File install.ps1 -Mode User
```

Then open `http://localhost:8080`. A container image and Debian packages are available as well.

## Documentation

The [documentation](https://barthjs.github.io/photopi/) describes the installation, the use, and the development
of PhotoPi. Every installation also serves the documentation of its own version under `/docs/`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

PhotoPi is licensed under the [MIT License](LICENSE).
