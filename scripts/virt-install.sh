virt-install --name oe2203sp1 \
  --memory 65536 --vcpus 32 --cpu host-passthrough \
  --machine virt --boot uefi,cdrom,hd \
  --disk path=/var/lib/libvirt/images/oe2203sp1.qcow2,size=100,format=qcow2,bus=virtio \
  --cdrom /root/iso/openEuler-22.03-LTS-SP1-everything-aarch64-dvd.iso \
  --network network=default,model=virtio \
  --graphics vnc,listen=0.0.0.0 --video virtio \
  --noautoconsole