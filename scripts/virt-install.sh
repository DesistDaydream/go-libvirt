qemu-img create -f qcow2 /var/lib/libvirt/images/openeuler-2203-aarch64.qcow2 100G

virt-install --name openeuler-2203-aarch64 \
  --memory 65536 --vcpus 32 --cpu host-passthrough \
  --machine virt --boot uefi,cdrom,hd \
  --os-variant openeuler22.03 \
  --disk path=/var/lib/libvirt/images/openeuler-2203-aarch64.qcow2,format=qcow2,bus=virtio \
  --cdrom /opt/iso/openEuler-22.03-LTS-SP1-everything-aarch64-dvd.iso \
  --network bridge=br0,model=virtio \
  --graphics vnc,listen=0.0.0.0 --video virtio \
  --noautoconsole