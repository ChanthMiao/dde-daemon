/**
 * Copyright (C) 2014 Deepin Technology Co., Ltd.
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 3 of the License, or
 * (at your option) any later version.
 **/

package audio

import (
	"fmt"
	"gir/gio-2.0"
	"pkg.deepin.io/dde/api/soundutils"
	. "pkg.deepin.io/dde/daemon/loader"
	"pkg.deepin.io/lib/dbus"
	"pkg.deepin.io/lib/log"
	"pkg.deepin.io/lib/pulse"
	"sync"
)

type Audio struct {
	init bool
	core *pulse.Context

	// 输出设备 ObjectPath 列表
	Sinks []*Sink
	// 输入设备ObjectPath 列表
	Sources []*Source
	// 正常输出声音的程序列表
	SinkInputs []*SinkInput

	Cards string

	// 默认的输出设备名称
	DefaultSink string
	// 默认的输入设备名称
	DefaultSource string

	// 最大音量
	MaxUIVolume float64

	cards CardInfos

	siEventChan  chan func()
	siPollerExit chan struct{}

	isSaving    bool
	saverLocker sync.Mutex

	sinkLocker sync.Mutex
}

const (
	audioSchema       = "com.deepin.dde.audio"
	gsKeyFirstRun     = "first-run"
	gsKeyInputVolume  = "input-volume"
	gsKeyOutputVolume = "output-volume"
)

var (
	defaultInputVolume  float64
	defaultOutputVolume float64
)

func (a *Audio) Reset() {
	for _, s := range a.Sinks {
		s.SetVolume(defaultOutputVolume, false)
		s.SetBalance(0, false)
		s.SetFade(0)
	}
	for _, s := range a.Sources {
		s.SetVolume(defaultInputVolume, false)
		s.SetBalance(0, false)
		s.SetFade(0)
	}
}

func (s *Audio) GetDefaultSink() *Sink {
	for _, o := range s.Sinks {
		if o.Name == s.DefaultSink {
			return o
		}
	}
	return nil
}
func (s *Audio) GetDefaultSource() *Source {
	for _, o := range s.Sources {
		if o.Name == s.DefaultSource {
			return o
		}
	}
	return nil
}

// SetProfile activate the profile for the special card and disable others card
// The available sinks and sources will also change with the profile changing.
func (a *Audio) SetProfile(cardId uint32, profile string) error {
	var (
		others []*pulse.Card

		hasActivatedCard bool = false
	)

	for _, card := range a.core.GetCardList() {
		if card.Index != cardId {
			others = append(others, card)
			continue
		}

		if !(cProfileInfos2(card.Profiles).exist(profile)) {
			return fmt.Errorf("Invalid profile '%s' for %s", profile, card.Name)
		}
		if card.ActiveProfile.Name != profile {
			card.SetProfile(profile)
		}
		hasActivatedCard = true
	}

	if !hasActivatedCard {
		return fmt.Errorf("Invalid card id: %v", cardId)
	}

	// disable other card, otherwise the active card maybe not work
	for _, card := range others {
		card.SetProfile("off")
	}

	return nil
}

func NewSink(core *pulse.Sink) *Sink {
	s := &Sink{core: core}
	s.index = s.core.Index
	s.update()
	return s
}
func NewSource(core *pulse.Source) *Source {
	s := &Source{core: core}
	s.index = s.core.Index
	s.update()
	return s
}
func NewSinkInput(core *pulse.SinkInput) *SinkInput {
	s := &SinkInput{core: core}
	s.index = s.core.Index
	s.update()
	return s
}
func NewAudio(core *pulse.Context) *Audio {
	a := &Audio{core: core}
	a.MaxUIVolume = pulse.VolumeUIMax
	a.siEventChan = make(chan func(), 10)
	a.siPollerExit = make(chan struct{})
	a.applyConfig()
	a.update()
	a.initEventHandlers()

	go a.sinkInputPoller()

	return a
}

func (a *Audio) destroy() {
	close(a.siPollerExit)
	dbus.UnInstallObject(a)
}

func (a *Audio) SetDefaultSink(name string) {
	a.sinkLocker.Lock()
	defer a.sinkLocker.Unlock()

	a.core.SetDefaultSink(name)
	a.update()
	a.saveConfig()

	var idxList []uint32
	for _, sinkInput := range a.SinkInputs {
		idxList = append(idxList, sinkInput.index)
	}
	if len(idxList) == 0 {
		return
	}
	a.core.MoveSinkInputsByName(idxList, name)
}
func (a *Audio) SetDefaultSource(name string) {
	a.core.SetDefaultSource(name)
	a.update()
	a.saveConfig()
}

type Port struct {
	Name        string
	Description string
	Available   byte // Unknow:0, No:1, Yes:2
}
type Sink struct {
	core  *pulse.Sink
	index uint32

	Name        string
	Description string

	// 默认音量值
	BaseVolume float64

	// 是否静音
	Mute bool

	// 当前音量
	Volume float64
	// 左右声道平衡值
	Balance float64
	// 是否支持左右声道调整
	SupportBalance bool
	// 前后声道平衡值
	Fade float64
	// 是否支持前后声道调整
	SupportFade bool

	// 支持的输出端口
	Ports []Port
	// 当前使用的输出端口
	ActivePort Port
}

// 设置音量大小
//
// v: 音量大小
//
// isPlay: 是否播放声音反馈
func (s *Sink) SetVolume(v float64, isPlay bool) error {
	if !isVolumeValid(v) {
		return fmt.Errorf("Invalid volume value: %v", v)
	}

	if v == 0 {
		v = 0.001
	}
	s.core.SetVolume(s.core.Volume.SetAvg(v))
	if isPlay {
		playFeedbackWithDevice(s.Name)
	}
	return nil
}

// 设置左右声道平衡值
//
// v: 声道平衡值
//
// isPlay: 是否播放声音反馈
func (s *Sink) SetBalance(v float64, isPlay bool) error {
	if v < -1.00 || v > 1.00 {
		return fmt.Errorf("Invalid volume value: %v", v)
	}

	s.core.SetVolume(s.core.Volume.SetBalance(s.core.ChannelMap, v))
	if isPlay {
		playFeedbackWithDevice(s.Name)
	}
	return nil
}

// 设置前后声道平衡值
//
// v: 声道平衡值
//
// isPlay: 是否播放声音反馈
func (s *Sink) SetFade(v float64) error {
	if v < -1.00 || v > 1.00 {
		return fmt.Errorf("Invalid volume value: %v", v)
	}

	s.core.SetVolume(s.core.Volume.SetFade(s.core.ChannelMap, v))
	playFeedbackWithDevice(s.Name)
	return nil
}

// 是否静音
func (s *Sink) SetMute(v bool) {
	s.core.SetMute(v)
	if !v {
		playFeedbackWithDevice(s.Name)
	}
}

// 设置此设备的当前使用端口
func (s *Sink) SetPort(name string) {
	s.core.SetPort(name)
}

type SinkInput struct {
	core  *pulse.SinkInput
	index uint32

	// process name
	Name string
	Icon string
	Mute bool

	Volume         float64
	Balance        float64
	SupportBalance bool
	Fade           float64
	SupportFade    bool
}

func (s *SinkInput) SetVolume(v float64, isPlay bool) error {
	if !isVolumeValid(v) {
		return fmt.Errorf("Invalid volume value: %v", v)
	}

	if v == 0 {
		v = 0.001
	}
	s.core.SetVolume(s.core.Volume.SetAvg(v))
	if isPlay {
		playFeedback()
	}
	return nil
}
func (s *SinkInput) SetBalance(v float64, isPlay bool) error {
	if v < -1.00 || v > 1.00 {
		return fmt.Errorf("Invalid volume value: %v", v)
	}

	s.core.SetVolume(s.core.Volume.SetBalance(s.core.ChannelMap, v))
	if isPlay {
		playFeedback()
	}
	return nil
}
func (s *SinkInput) SetFade(v float64) error {
	if v < -1.00 || v > 1.00 {
		return fmt.Errorf("Invalid volume value: %v", v)
	}

	s.core.SetVolume(s.core.Volume.SetFade(s.core.ChannelMap, v))
	playFeedback()
	return nil
}
func (s *SinkInput) SetMute(v bool) {
	s.core.SetMute(v)
	if !v {
		playFeedback()
	}
}

type Source struct {
	core  *pulse.Source
	index uint32

	Name        string
	Description string

	// 默认的输入音量
	BaseVolume float64

	Mute bool

	Volume         float64
	Balance        float64
	SupportBalance bool
	Fade           float64
	SupportFade    bool

	Ports      []Port
	ActivePort Port
}

// 如何反馈输入音量？
func (s *Source) SetVolume(v float64, isPlay bool) error {
	if !isVolumeValid(v) {
		return fmt.Errorf("Invalid volume value: %v", v)
	}

	if v == 0 {
		v = 0.001
	}
	s.core.SetVolume(s.core.Volume.SetAvg(v))
	if isPlay {
		playFeedback()
	}
	return nil
}
func (s *Source) SetBalance(v float64, isPlay bool) error {
	if v < -1.00 || v > 1.00 {
		return fmt.Errorf("Invalid volume value: %v", v)
	}

	s.core.SetVolume(s.core.Volume.SetBalance(s.core.ChannelMap, v))
	if isPlay {
		playFeedback()
	}
	return nil
}
func (s *Source) SetFade(v float64) error {
	if v < -1.00 || v > 1.00 {
		return fmt.Errorf("Invalid volume value: %v", v)
	}

	s.core.SetVolume(s.core.Volume.SetFade(s.core.ChannelMap, v))
	playFeedback()
	return nil
}
func (s *Source) SetMute(v bool) {
	s.core.SetMute(v)
	if !v {
		playFeedback()
	}
}
func (s *Source) SetPort(name string) {
	s.core.SetPort(name)
}

type Daemon struct {
	*ModuleBase
}

func NewAudioDaemon(logger *log.Logger) *Daemon {
	var d = new(Daemon)
	d.ModuleBase = NewModuleBase("audio", d, logger)
	return d
}

func (*Daemon) GetDependencies() []string {
	return []string{}
}

var _audio *Audio

func finalize() {
	_audio.destroy()
	_audio = nil
	logger.EndTracing()
}

func (*Daemon) Start() error {
	if _audio != nil {
		return nil
	}

	logger.BeginTracing()

	ctx := pulse.GetContext()
	_audio = NewAudio(ctx)

	if err := dbus.InstallOnSession(_audio); err != nil {
		logger.Error("Failed InstallOnSession:", err)
		finalize()
		return err
	}

	initDefaultVolume(_audio)
	return nil
}

func (*Daemon) Stop() error {
	if _audio == nil {
		return nil
	}

	finalize()
	return nil
}

func playFeedback() {
	playFeedbackWithDevice("")
}

func playFeedbackWithDevice(device string) {
	soundutils.PlaySystemSound(soundutils.EventVolumeChanged, device, false)
}

func isVolumeValid(v float64) bool {
	if v < 0 || v > pulse.VolumeUIMax {
		return false
	}
	return true
}

func initDefaultVolume(audio *Audio) {
	setting := gio.NewSettings(audioSchema)
	defer setting.Unref()

	inVolumePer := float64(setting.GetInt(gsKeyInputVolume)) / 100.0
	outVolumePer := float64(setting.GetInt(gsKeyOutputVolume)) / 100.0
	defaultInputVolume = pulse.VolumeUIMax * inVolumePer
	defaultOutputVolume = pulse.VolumeUIMax * outVolumePer

	if !setting.GetBoolean(gsKeyFirstRun) {
		return
	}

	setting.SetBoolean(gsKeyFirstRun, false)
	for _, s := range audio.Sinks {
		s.SetVolume(defaultOutputVolume, false)
	}

	for _, s := range audio.Sources {
		s.SetVolume(defaultInputVolume, false)
	}
}
