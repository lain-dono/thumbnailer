# Thumbnailer Service

Оно использует systemd. Надо доустановить [rkt](https://coreos.com/rkt/).

Как установить всё это дело? А вот так:

```bash
# ./build.sh
# cp thumbnailer-linux-amd64.aci /opt/
# cp thumbnailer.service /etc/systemd/system/
```

Билдскрипт сам скачивает необходимые зависимости. А именно:
* [Go](https://golang.org/), если оного вдруг нет
* [acbuild](https://github.com/appc/acbuild), который требуется джля создания образа
* Статически слинкованный [ffmpeg](http://johnvansickle.com/ffmpeg/)

После запуска (`systemctl start thumbnailer.service`)
оно будет отсвечивать на порту 5000 и ждать POST-запросов.

Пост-запросы в `multipart/form-data`.

Параметр `file` это собсна webm, которую будем пилить.

Параметры `w` и `h` определяют размеры пикчи.
Оба по умолчанию 200 пикселей.

Параметр `interp` определяет алгоритм уменьшения размера.
По умолчанию NearestNeighbor.
Если передать невалидный, то не будет ресайзить вообще (это не баг, а фича).
Возможные значения такие (не зависит от регистра):
* NearestNeighbor
* Bilinear
* Bicubic
* MitchellNetravali
* Lanczos2
* Lanczos3

Алсо если зайти по [http://localhost:5000/form](http://localhost:5000/form),
то покажет пример формы, которой постить.

Логи смотреть так (логирует только ошибки): `journalctl -u thumbnailer.service`

