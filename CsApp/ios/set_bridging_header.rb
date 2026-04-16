#!/usr/bin/env ruby
require 'xcodeproj'

project_path = File.join(__dir__, 'CsApp.xcodeproj')
project = Xcodeproj::Project.open(project_path)
target = project.targets.find { |t| t.name == 'CsApp' }

# Add bridging header to project
group = project.main_group.find_subpath('CsApp', true)
unless group.files.any? { |f| f.path&.include?('Bridging-Header') }
  ref = group.new_file('CsApp/CsApp-Bridging-Header.h')
  puts "Added bridging header to project"
end

# Set build setting
target.build_configurations.each do |config|
  config.build_settings['SWIFT_OBJC_BRIDGING_HEADER'] = 'CsApp/CsApp-Bridging-Header.h'
  puts "Set SWIFT_OBJC_BRIDGING_HEADER in #{config.name}"
end

project.save
puts "Saved"
