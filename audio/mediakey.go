package main

import libsound "dbus/com/deepin/api/sound"
import "dbus/com/deepin/daemon/keybinding"

func (audio *Audio) listenMediaKey() {
	mediakey, err := keybinding.NewMediaKey("com.deepin.daemon.KeyBinding", "/com/deepin/daemon/MediaKey")
	if err != nil {
		Logger.Error("Can't create keybinding.MediaKey! Mediakey support will be disabled", err)
		return
	}

	player, err := libsound.NewSound("com.deepin.api.Sound", "/com/deepin/api/Sound")
	if err != nil {
		Logger.Error("Can't create com.deepin.api.Sound! Sound feedback support will be disabled", err)
	}

	mediakey.ConnectAudioMute(func(pressed bool) {
		if !pressed {
			sink := audio.GetDefaultSink()
			sink.SetSinkMute(!sink.Mute)
			player.PlaySystemSound("audio-volume-change")
		}
	})
	mediakey.ConnectAudioUp(func(pressed bool) {
		if !pressed {
			sink := audio.GetDefaultSink()
			volume := int32(sink.Volume + _VOLUME_STEP)
			if volume < 0 {
				volume = 0
			} else if volume > 100 {
				volume = 100
			}
			if sink.Volume < 100 {
				sink.setSinkVolume(uint32(volume))
				sink.setSinkMute(false)
			}
			player.PlaySystemSound("audio-volume-change")
		}
	})
	mediakey.ConnectAudioDown(func(pressed bool) {
		if !pressed {
			sink := audio.GetDefaultSink()
			volume := int32(sink.Volume - _VOLUME_STEP)
			if volume < 0 {
				volume = 0
			}
			sink.setSinkVolume(uint32(volume))
			sink.setSinkMute(false)
			player.PlaySystemSound("audio-volume-change")
		}
	})
}
