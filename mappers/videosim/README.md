# videosim — mapper DMI de cámara simulada

Mapper real de KubeEdge (framework oficial [mapper-framework](https://github.com/kubeedge/mapper-framework),
rama `release-1.23`): decodifica un `.mp4` en loop con ffmpeg y corre detección de movimiento real por
diferencia de frames. Reporta los twins de `camera-01` vía DMI (gRPC) a EdgeCore, que los sube por el
pipeline nativo hasta el CRD `DeviceStatus` en el cloud.

> La historia completa de cómo se construyó y los problemas encontrados (RBAC del chart upstream roto,
> membresía que nunca llegaba al edge, video sin movimiento por comillas rotas, etc.) está en la sección 7
> de la BITACORA.md del repo [kubeedge](https://github.com/enriquesoto/kubeedge).

Este directorio guarda **solo lo que escribimos nosotros** (driver + config); el resto es el template
generado por el framework.

## Regenerar el proyecto y compilar (en el nodo edge, Go >= 1.23)

```bash
git clone --depth 1 -b release-1.23 https://github.com/kubeedge/mapper-framework.git
git clone --depth 1 -b release-1.23 https://github.com/kubeedge/api.git
# generación manual equivalente a `make generate` (nombre: videosim, método: nostream)
cp -r mapper-framework/_template/mapper videosim
cd videosim/data/stream && ls | grep -v handler_nostream.go | xargs rm -rf && mv handler_nostream.go handler.go && cd -
grep -rl Template videosim | xargs sed -i "s/Template/Videosim/g"
grep -rl "kubeedge/Videosim" videosim | xargs sed -i "s/kubeedge\/Videosim/kubeedge\/videosim/g"
# sobrescribir con nuestros archivos y compilar
cp <este-dir>/driver/*.go videosim/driver/ && cp <este-dir>/config.yaml videosim/config.yaml
cd videosim && go mod tidy && go build -o videosim-mapper ./cmd/main.go
```

## Despliegue (systemd en la VM edge)

- Binario: `/usr/local/bin/videosim-mapper`, config: `/etc/videosim/config.yaml`
- Unit: `videosim-mapper.service` con `After=edgecore.service` y `Restart=always`
- Video de prueba `/var/lib/videosim/sample.mp4` (loop 20s: 10s estático + 10s animado):
  ```bash
  ffmpeg -f lavfi -i color=c=gray:s=640x480:d=10:r=10 \
         -f lavfi -i testsrc2=s=640x480:d=10:r=10 \
         -filter_complex "[0][1]concat=n=2:v=1:a=0[out]" -map "[out]" -pix_fmt yuv420p sample.mp4
  ```

## Verlo funcionando

```bash
kubectl get devicestatus camera-01 -n kubeedge \
  -o jsonpath='{range .status.twins[*]}{.propertyName}{" = "}{.reported.value}{"\n"}{end}'
```

`motionDetected`/`confidence` alternan cada ~10s siguiendo las fases del video en loop.
