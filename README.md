# Thumbnailer Service

Оно использует systemd.

Как установить всё это дело? А вот так:

```bash
# ./build.sh
# cp thumbnailer-linux-amd64.aci /opt/
# cp thumbnailer.service /etc/systemd/system/
```

После запуска (`systemctl start thumbnailer.service`)
оно будет отсвечивать на порту 5000 и ждать POST-запросов.

Пост-запросы в `multipart/form-data`.

Параметр `file` это собсна webm, которую будем пилить.

Параметры `w` и `h` определяют размеры пикчи.
Оба по умолчанию 200 пикселей.

Параметр `interp` определяет алгоритм уменьшения размера.
По умолчанию NearestNeighbor.
Если передать невалидный, то не будет ресайзить вообще.
(Это не баг, а фича)
Возможные значения такие:
* resize.NearestNeighbor)
* Bilinear
* Bicubic
* MitchellNetravali
* Lanczos2
* Lanczos3

Алсо если зайти по http://localhost:5000/form
то покажет пример формы, которой постить.

